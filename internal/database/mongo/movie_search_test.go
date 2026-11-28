package mongo

import (
	"regexp"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

func TestMovieSearchTitleAndYearAreIndependent(t *testing.T) {
	filter := movieSearchFilter("Hunger Games", "2023")
	parts := filter.Map()["$and"].(bson.A)
	pattern := func(part interface{}) *regexp.Regexp {
		return regexp.MustCompile(part.(bson.D).Map()["file_name"].(bson.D).Map()["$regex"].(string))
	}
	title, year := pattern(parts[0]), pattern(parts[1])
	for _, tc := range []struct {
		name string
		want bool
	}{
		{"The.Hunger.Games.The.Ballad.of.Songbirds.and.Snakes.2023.720p.mkv", true},
		{"The Hunger Games 2023 Tamil Dubbed", true},
		{"HungerGames_(2023)_Tamil.mkv", true},
		{"The.Hunger.Games.2012.Tamil.mkv", false},
		{"Hunger.Games.20230.mkv", false},
		{"Another.Movie.2023.Tamil.mkv", false},
	} {
		if got := title.MatchString(tc.name) && year.MatchString(tc.name); got != tc.want {
			t.Errorf("%q matched = %v, want %v", tc.name, got, tc.want)
		}
	}
	if _, exists := movieSearchFilter("Hunger Games", "").Map()["$and"]; exists {
		t.Fatal("title-only search should have no year constraint")
	}
}
