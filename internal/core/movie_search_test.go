package core

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/Jisin0/autofilterbot/internal/autofilter"
	"github.com/Jisin0/autofilterbot/internal/model"
)

type movieFixtureCursor struct {
	rows []model.File
	next int
}

func (c *movieFixtureCursor) Next(context.Context) bool {
	if c.next == len(c.rows) {
		return false
	}
	c.next++
	return true
}

func (c *movieFixtureCursor) Decode(out interface{}) error {
	*out.(*model.File) = c.rows[c.next-1]
	return nil
}

func (c *movieFixtureCursor) Close(context.Context) error { return nil }

func TestMovieAudioSearchCombinations(t *testing.T) {
	makeFile := func(id, name, caption string) model.File {
		return model.File{UniqueId: id, FileName: name, FileType: model.FileTypeDocument, Caption: &caption}
	}
	sequel := "The.Hunger.Games.The.Ballad.of.Songbirds.and.Snakes.2023"
	original := makeFile("old", "The.Hunger.Games.2012.mkv", "Tamil Dubbed")
	releases := []model.File{
		makeFile("multi", sequel+".720p.mkv", "Audio: Tamil, Hindi, Telugu\nSubtitles: English, Malayalam"),
		makeFile("english", sequel+".1080p.mkv", "English audio | Tamil subtitles"),
		makeFile("malayalam", sequel+".4K.mkv", "Malayalam Audio | English Sub"),
		makeFile("kannada", sequel+".BluRay.mkv", "Dubbed in: Kannada"),
		{UniqueId: "legacy-tag", FileName: sequel + ".WEBRip.mkv"},
		{UniqueId: "legacy-name", FileName: sequel + ".English.AAC.mkv"},
		makeFile("subtitle", sequel+".Tamil.srt", "Tamil"),
		makeFile("edited", sequel+".Proper.mkv", "Subtitles: Tamil"),
	}
	all := append(slices.Clone(releases), original)
	legacy := func(ids []string) (map[string]bool, error) {
		if slices.Contains(ids, "edited") {
			t.Error("edited caption must take precedence over the legacy tag")
		}
		return map[string]bool{"legacy-tag": true}, nil
	}

	for _, tc := range []struct {
		query, year, audio string
		want               []string
	}{
		{"Hunger Games", "", "", []string{"multi", "english", "malayalam", "kannada", "legacy-tag", "legacy-name", "subtitle", "edited", "old"}},
		{"Hunger Games 2023", "2023", "", []string{"multi", "english", "malayalam", "kannada", "legacy-tag", "legacy-name", "subtitle", "edited"}},
		{"Hunger Games Tamil", "", "tam", []string{"multi", "legacy-tag", "old"}},
		{"Hunger Games 2023 Tamil", "2023", "tam", []string{"multi", "legacy-tag"}},
		{"Hunger Games Tamil Audio 2023", "2023", "tam", []string{"multi", "legacy-tag"}},
		{"Hunger Games 2023 Hindi", "2023", "hin", []string{"multi"}},
		{"Hunger Games 2023 Malayalam", "2023", "mal", []string{"malayalam"}},
		{"Hunger Games 2023 Telugu", "2023", "tel", []string{"multi"}},
		{"Hunger Games 2023 Kannada", "2023", "kan", []string{"kannada"}},
		{"Hunger Games 2023 English", "2023", "eng", []string{"english", "legacy-name"}},
		{"Hunger Games 2012 Hindi", "2012", "hin", nil},
	} {
		t.Run(tc.query, func(t *testing.T) {
			parsed := parseMovieSearch(tc.query)
			if parsed != (movieSearch{title: "Hunger Games", year: tc.year, audio: tc.audio}) {
				t.Fatalf("parsed query %+v", parsed)
			}
			rows := all
			switch parsed.year {
			case "2023":
				rows = releases
			case "2012":
				rows = []model.File{original}
			}
			cursor := &movieFixtureCursor{rows: rows}
			opts := fullSeriesSearchOptions{perPage: 2}
			var pages []autofilter.Files
			var err error
			if parsed.audio == "" {
				pages, err = autofilter.FilesFromCursor(context.Background(), cursor, opts)
			} else {
				pages, err = audioSearchFiles(context.Background(), cursor, opts, parsed.audio, legacy)
			}
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, page := range pages {
				for _, file := range page {
					got = append(got, file.UniqueId)
				}
			}
			if !slices.Equal(got, tc.want) {
				t.Fatalf("files = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAudioSearchFiltersBeforeLimit(t *testing.T) {
	rows := make([]model.File, 502)
	caption := "Thai Audio | Tamil Sub"
	for i := range rows {
		rows[i] = model.File{UniqueId: fmt.Sprint(i), FileName: "Hunger.Games.2023.mkv", Caption: &caption}
	}
	rows[501].Caption = nil
	rows[501].OriginalFileName = "Hunger.Games.2023.Tamil.Dub.mkv"
	pages, err := audioSearchFiles(context.Background(), &movieFixtureCursor{rows: rows},
		fullSeriesSearchOptions{perPage: 10}, "tam", func([]string) (map[string]bool, error) { return nil, nil })
	if err != nil || len(pages) != 1 || len(pages[0]) != 1 || pages[0][0].UniqueId != "501" {
		t.Fatalf("pages = %+v, err = %v", pages, err)
	}
}
