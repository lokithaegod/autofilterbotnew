package mongo

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/Jisin0/autofilterbot/internal/database"
	"github.com/Jisin0/autofilterbot/internal/functions"
	"github.com/Jisin0/autofilterbot/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (c *Client) SaveFile(f *model.File) error {
	// Find any with matching file_id
	byID := fileIdFilter(f.FileId)
	if err := c.fileCollection.FindOne(c.ctx, byID).Err(); err != mongo.ErrNoDocuments {
		if err == nil {
			return c.saveDuplicateCaption(f, byID)
		}
		return err
	}

	// Find a document that starts with the same file_name and is within a 100 byte range of file_size
	duplicateFilter := bson.D{
		{Key: "file_name", Value: bson.D{{Key: "$regex", Value: "^" + regexp.QuoteMeta(f.FileName) + "$"}}},
		{Key: "file_size", Value: bson.D{
			{Key: "$gte", Value: f.FileSize - 100},
			{Key: "$lte", Value: f.FileSize + 100},
		}},
	}
	if err := c.fileCollection.FindOne(c.ctx, duplicateFilter).Err(); err != mongo.ErrNoDocuments {
		if err == nil {
			// The approximate duplicate check remains in place. Copy metadata
			// only when the byte size is also an exact match.
			exact := bson.D{{Key: "file_name", Value: f.FileName}, {Key: "file_size", Value: f.FileSize}}
			return c.saveDuplicateCaption(f, exact)
		}
		return err
	}

	_, err := c.fileCollection.InsertOne(c.ctx, f)

	return err
}

func (c *Client) saveDuplicateCaption(f *model.File, identity bson.D) error {
	if f.Caption != nil {
		filter := append(identity, bson.E{Key: "caption", Value: bson.M{"$exists": false}})
		if _, err := c.fileCollection.UpdateMany(c.ctx, filter, captionUpdate(f.OriginalFileName, *f.Caption)); err != nil {
			return err
		}
	}
	return database.FileAlreadyExistsError{FileName: f.FileName}
}

func captionUpdate(filename, caption string) bson.M {
	fields := bson.M{"caption": caption}
	if filename != "" {
		fields["original_file_name"] = filename
	}
	return bson.M{"$set": fields}
}

// SetFileCaptionForMedia refreshes source metadata on existing rows only.
func (c *Client) SetFileCaptionForMedia(id, filename string, size int64, caption string) (int, error) {
	identities := bson.A{bson.M{"_id": id}}
	if filename != "" && size > 0 {
		identities = append(identities, bson.M{
			"file_name": bson.M{"$in": []string{filename, functions.RemoveSymbols(functions.RemoveExtension(filename))}},
			"file_size": size,
		})
	}
	ctx, cancel := context.WithTimeout(c.ctx, 12*time.Second)
	defer cancel()
	result, err := c.fileCollection.UpdateMany(ctx, bson.M{"$or": identities}, captionUpdate(filename, caption))
	return int(result.MatchedCount), err
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
	if len(words) == 0 {
		return "^$"
	}
	if len(words) > 20 {
		words = words[:20]
	}
	for i, w := range words {
		words[i] = regexp.QuoteMeta(w)
	}
	return "(?i)(^|[^a-z0-9])" + strings.Join(words, "[^a-z0-9]*") + "($|[^a-z0-9])"
}

// Fringe episodes have both "fri.s01e01..." and "Fringe.S04E13..."
// names. The alias is intentionally limited to the short episode prefix.
var fringeAlias = regexp.MustCompile(`(?i)^fringe(?:[ ._-]+s([0-9]{1,2})[ ._-]*e([0-9]{1,3}))?$`)

func fileSearchFilter(query string) bson.D {
	original := bson.D{{Key: "file_name", Value: bson.D{{Key: "$regex", Value: filenameSearchPattern(query)}}}}
	parts := fringeAlias.FindStringSubmatch(strings.TrimSpace(query))
	if parts == nil {
		return original
	}
	alias := `(?i)^fri[._ -]*s[0-9]{1,2}[ ._-]*e[0-9]{1,3}(?:[^a-z0-9]|$)`
	if parts[1] != "" {
		alias = "(?i)^fri[._ -]*s" + regexp.QuoteMeta(parts[1]) + "[ ._-]*e" + regexp.QuoteMeta(parts[2]) + "(?:[^a-z0-9]|$)"
	}
	return bson.D{{Key: "$or", Value: bson.A{
		original,
		bson.D{{Key: "file_name", Value: bson.D{{Key: "$regex", Value: alias}}}},
	}}}
}

func (c *Client) SearchFiles(query string) (database.Cursor, error) {
	return c.fileCollection.Find(context.Background(), fileSearchFilter(query),
		options.Find().SetSort(bson.M{"time": -1}).SetLimit(500))
}

// SearchMovieCandidates keeps the title pattern and release year independent
// so sequel subtitles may appear between them.
func (c *Client) SearchMovieCandidates(ctx context.Context, query, year string, filterAudio bool) (database.Cursor, error) {
	filter := movieSearchFilter(query, year)
	opts := options.Find().SetSort(bson.M{"time": -1})
	if !filterAudio {
		opts.SetLimit(500)
	}
	return c.fileCollection.Find(ctx, filter, opts)
}

func movieSearchFilter(query, year string) bson.D {
	title := fileSearchFilter(query)
	if year == "" {
		return title
	}
	return bson.D{{Key: "$and", Value: bson.A{title, bson.D{
		{Key: "file_name", Value: bson.D{{Key: "$regex", Value: filenameSearchPattern(year)}}},
	}}}}
}

// fileIdFilter creates a bson filter to match by file_id.
func fileIdFilter(id string) bson.D {
	return bson.D{{Key: "file_id", Value: id}}
}
