package mongo

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AvailabilityAlert struct {
	Generation      primitive.ObjectID `bson:"generation"`
	DeliveryStarted bool               `bson:"delivery_started"`
	ID              string             `bson:"_id"`
	UserID          int64              `bson:"user_id"`
	Title           string             `bson:"title"`
	Year            string             `bson:"year"`
	Audio           string             `bson:"audio"`
}

func (a AvailabilityAlert) key() string {
	return fmt.Sprintf("%d:%x", a.UserID, sha256.Sum256([]byte(a.Title+"\x00"+a.Year+"\x00"+a.Audio)))
}

const availabilityAlertsCollection = "AvailabilityAlerts"

func (c *Client) SaveAvailabilityAlert(ctx context.Context, a AvailabilityAlert) error {
	_, err := c.db.Collection(availabilityAlertsCollection).UpdateOne(ctx, bson.M{"_id": a.key()}, bson.M{"$setOnInsert": bson.M{
		"user_id": a.UserID, "title": a.Title, "year": a.Year, "audio": a.Audio, "next_check": time.Now(), "generation": primitive.NewObjectID(),
	}}, options.Update().SetUpsert(true))
	return err
}

func (c *Client) RemoveAvailabilityAlert(ctx context.Context, a AvailabilityAlert) (bool, error) {
	result, err := c.db.Collection(availabilityAlertsCollection).DeleteOne(ctx, bson.M{"_id": a.key()})
	return err == nil && result.DeletedCount == 1, err
}

func (c *Client) AvailabilityAlerts(ctx context.Context, userID int64, limit int64) ([]AvailabilityAlert, error) {
	filter := bson.M{"next_check": bson.M{"$lte": time.Now()}, "delivery_started": bson.M{"$ne": true}}
	if userID != 0 {
		filter = bson.M{"user_id": userID}
	}
	cursor, err := c.db.Collection(availabilityAlertsCollection).Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "next_check", Value: 1}, {Key: "_id", Value: 1}}).SetLimit(limit))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())
	var alerts []AvailabilityAlert
	err = cursor.All(ctx, &alerts)
	return alerts, err
}

func (a AvailabilityAlert) pendingFilter() bson.M {
	return bson.M{"_id": a.ID, "generation": a.Generation, "delivery_started": bson.M{"$ne": true}}
}

func (c *Client) DeferAvailabilityAlert(ctx context.Context, a AvailabilityAlert, next time.Time) error {
	_, err := c.db.Collection(availabilityAlertsCollection).UpdateOne(ctx, a.pendingFilter(), bson.M{"$set": bson.M{"next_check": next}})
	return err
}

// ClaimAvailabilityAlert records the attempt before contacting Telegram so
// overlapping processes and uncertain network responses cannot cause duplicate sends.
func (c *Client) ClaimAvailabilityAlert(ctx context.Context, a AvailabilityAlert) (bool, error) {
	result, err := c.db.Collection(availabilityAlertsCollection).UpdateOne(ctx, a.pendingFilter(), bson.M{"$set": bson.M{"delivery_started": true}})
	return err == nil && result.ModifiedCount == 1, err
}

func (c *Client) CompleteAvailabilityAlert(ctx context.Context, a AvailabilityAlert, retryAfter time.Duration, delivered bool) error {
	identity := bson.M{"_id": a.ID, "generation": a.Generation, "delivery_started": true}
	collection := c.db.Collection(availabilityAlertsCollection)
	if delivered {
		_, err := collection.DeleteOne(ctx, identity)
		return err
	}
	if retryAfter > 0 {
		_, err := collection.UpdateOne(ctx, identity, bson.M{"$set": bson.M{"delivery_started": false, "next_check": time.Now().Add(retryAfter)}})
		return err
	}
	return nil
}

func (c *Client) EnsureAvailabilityIndexes(ctx context.Context) error {
	_, err := c.db.Collection(availabilityAlertsCollection).Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "next_check", Value: 1}, {Key: "_id", Value: 1}}},
		{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "next_check", Value: 1}, {Key: "_id", Value: 1}}},
	})
	return err
}
