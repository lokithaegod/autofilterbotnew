package mongo

import (
 "fmt"
 "go.mongodb.org/mongo-driver/bson"
 "go.mongodb.org/mongo-driver/mongo"
 "go.mongodb.org/mongo-driver/mongo/options"
)

// GetIndexCheckpoint returns the last completely scanned message id for a channel.
// An absent record means that the channel has not used incremental indexing yet.
func (c *Client) GetIndexCheckpoint(channelID int64) (int64, error) {
 var v struct {Last int64 `bson:"last"`}
 err := c.db.Collection("IndexCheckpoints").FindOne(c.ctx, bson.M{"_id":channelID}).Decode(&v)
 if err == mongo.ErrNoDocuments { return 0,nil }
 return v.Last,err
}

// AdvanceIndexCheckpoint only advances the cursor, including if an older job finishes late.
func (c *Client) AdvanceIndexCheckpoint(channelID, last int64) error {
 if last < 1 {return fmt.Errorf("invalid checkpoint: %d",last)}
 _,err := c.db.Collection("IndexCheckpoints").UpdateOne(c.ctx,bson.M{"_id":channelID},
 bson.M{"$max":bson.M{"last":last}},options.Update().SetUpsert(true))
 return err
}
