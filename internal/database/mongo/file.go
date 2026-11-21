package mongo

import (
	"context"
	"regexp"
	"strings"

	"github.com/Jisin0/autofilterbot/internal/database"
	"github.com/Jisin0/autofilterbot/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (c *Client) SaveFile(f *model.File) error {
	// Find any with matching file_id
	if err := c.fileCollection.FindOne(c.ctx, fileIdFilter(f.FileId)).Err(); err != mongo.ErrNoDocuments {
		if err == nil { return database.FileAlreadyExistsError{FileName: f.FileName} }
		return err
	}

	// Find a document that starts with the same file_name and is within a 100 byte range of file_size
	duplicateFilter := bson.D{
		{Key: "file_name", Value: bson.D{{Key: "$regex", Value: "^" + f.FileName}}},
		{Key: "file_size", Value: bson.D{
			{Key: "$gte", Value: f.FileSize - 100},
			{Key: "$lte", Value: f.FileSize + 100},
		}},
	}
	if err := c.fileCollection.FindOne(c.ctx, duplicateFilter).Err(); err != mongo.ErrNoDocuments {
		if err == nil { return database.FileAlreadyExistsError{FileName: f.FileName} }
		return err
	}

	_, err := c.fileCollection.InsertOne(c.ctx, f)

	return err
}

func (c *Client) SaveFiles(files ...*model.File) []error {
	var errs []error

	for _, f := range files {
		if err := c.SaveFile(f); err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

func (c *Client) GetFile(fileId string) (*model.File, error) {
	res := c.fileCollection.FindOne(c.ctx, idFilter(fileId))
	if err := res.Err(); err != nil {
		return nil, err
	}

	var f model.File

	err := res.Decode(&f)

	return &f, err
}

func (c *Client) DeleteFile(fileId string) error {
	_, err := c.fileCollection.DeleteOne(c.ctx, idFilter(fileId))
	return err
}

// filenameSearchPattern makes natural text searches match separator-heavy filenames.
// It tolerates joined words (TimeIn), dots, spaces, hyphens, and underscores.
// Tokens are escaped to keep search input as text rather than regex syntax.
func filenameSearchPattern(query string) string {
    words := strings.Fields(strings.ToLower(query))
    if len(words) == 0 {return "^$"}
    if len(words) > 20 {words = words[:20]}
    for i,w := range words {words[i] = regexp.QuoteMeta(w)}
    return "(?i)(^|[^a-z0-9])" + strings.Join(words, "[^a-z0-9]*") + "($|[^a-z0-9])"
}

func (c *Client) SearchFiles(query string) (database.Cursor, error) {
    pattern := filenameSearchPattern(query)
    filter := bson.D{{Key: "file_name", Value: bson.D{{Key: "$regex", Value: pattern}}}}
    return c.fileCollection.Find(context.Background(), filter, options.Find().SetSort(bson.M{"time": -1}).SetLimit(50))
}

// fileIdFilter creates a bson filter to match by file_id.
func fileIdFilter(id string) bson.D {
	return bson.D{{Key: "file_id", Value: id}}
}
