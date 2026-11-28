package mongo

import (
	"context"

	"github.com/Jisin0/autofilterbot/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// VisitCaptionAuditFiles reads only audit fields from each configured file collection.
func (c *Client) VisitCaptionAuditFiles(ctx context.Context, title, year string, visit func(model.File)) error {
	filter := bson.D{}
	if title != "" {
		filter = movieSearchFilter(title, year)
	}
	projection := bson.M{"file_name": 1, "original_file_name": 1, "caption": 1, "file_type": 1, "file_link": 1}
	for _, collection := range c.fileCollection.allCollections {
		cursor, err := collection.Find(ctx, filter, options.Find().SetProjection(projection))
		if err != nil {
			return err
		}
		err = func() error {
			defer cursor.Close(context.Background())
			for cursor.Next(ctx) {
				var file model.File
				if err := cursor.Decode(&file); err != nil {
					return err
				}
				visit(file)
			}
			return cursor.Err()
		}()
		if err != nil {
			return err
		}
	}
	return nil
}
