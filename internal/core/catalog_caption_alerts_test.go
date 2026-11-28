package core

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/Jisin0/autofilterbot/internal/autofilter"
	"github.com/Jisin0/autofilterbot/internal/model"
)

func TestFindUsesTypedSearchFilters(t *testing.T) {
	for _, query := range []string{"Hunger Games 2023 Telugu", "Hunger Games year:2023 lang:tel", "Hunger Games Telugu year:2023"} {
		got, err := parseDiscover(query)
		if err != nil || got.Query != "Hunger Games" || got.Year != "2023" || got.Language != "tel" {
			t.Fatalf("%q parsed as %+v, %v", query, got, err)
		}
	}
}

func TestCatalogAndButtonsShareAudioClassification(t *testing.T) {
	audio, subs := "Audio: Telugu | Subtitles: Kannada", "Subtitle: Telugu"
	rows := []model.File{
		{UniqueId: "video", FileName: "Film.2023.mkv", Caption: &audio},
		{UniqueId: "only-subs", FileName: "Film.2023.1080p.mkv", Caption: &subs},
		{UniqueId: "document", FileName: "Film 2023 Telugu", OriginalFileName: "Film.2023.Telugu.srt"},
		{UniqueId: "legacy", FileName: "Film.2023.720p.mkv"},
	}
	tags := func(ids []string) (map[string]bool, error) {
		if !slices.Contains(ids, "legacy") {
			t.Error("legacy Tamil tag was not requested")
		}
		return map[string]bool{"legacy": true}, nil
	}
	pages, err := collectSearchFiles(context.Background(), &movieFixtureCursor{rows: rows}, 10, 10, "", tags, false)
	if err != nil {
		t.Fatal(err)
	}
	getIDs := func(files autofilter.Files) []string {
		var ids []string
		for _, f := range files {
			ids = append(ids, f.UniqueId)
		}
		return ids
	}
	for _, tt := range []struct {
		code, kind string
		want       []string
	}{
		{"tel", "all", []string{"video"}},
		{"kan", "all", nil},
		{"tam", "all", []string{"legacy"}},
		{"tel", "sub", []string{"document"}},
	} {
		if got := getIDs(matchingSubtitleFiles(pages, "all", tt.code, "all", tt.kind)); !slices.Equal(got, tt.want) {
			t.Errorf("%s/%s got %v, want %v", tt.code, tt.kind, got, tt.want)
		}
	}
	filtered, err := collectSearchFiles(context.Background(), &movieFixtureCursor{rows: rows}, 10, 10, "tel", tags, false)
	if err != nil || len(filtered) != 1 || !slices.Equal(getIDs(filtered[0]), []string{"video"}) {
		t.Fatalf("catalog filtered = %v, %v", filtered, err)
	}
}

func TestAvailabilitySkipsSubtitleCandidatesBeforeLimit(t *testing.T) {
	rows := []model.File{
		{UniqueId: "subs", FileName: "Film.2023.srt"},
		{UniqueId: "video", FileName: "Film.2023.mkv"},
	}
	pages, err := collectSearchFiles(context.Background(), &movieFixtureCursor{rows: rows}, 1, 1, "",
		func([]string) (map[string]bool, error) { return nil, nil }, true)
	if err != nil || len(pages) != 1 || pages[0][0].UniqueId != "video" {
		t.Fatalf("unexpected alert matches: %v (%v)", pages, err)
	}
}

func TestCaptionAuditSeparatesMissingLabelsAndSubtitleDocuments(t *testing.T) {
	var audit captionAudit
	caption := "English audio | Tamil subtitles"
	empty := ""
	for _, file := range []model.File{
		{FileName: "Film.2023.mkv", Caption: &caption},
		{FileName: "Film.2023.Tamil.mkv"},
		{FileName: "Film.2023.720p.mkv", MessageLink: "https://t.me/c/123/456"},
		{FileName: "Film.2023.1080p.mkv", Caption: &empty},
		{FileName: "Old upload 2023"},
		{FileName: "Film.Tamil.srt", Caption: &caption},
	} {
		audit.add(file)
	}
	if audit.videos != 4 || audit.saved != 2 || audit.labeled != 2 || audit.needsReview != 1 || audit.unknown != 1 {
		t.Fatalf("unexpected audit: %+v", audit)
	}
	if len(audit.samples) != 2 || audit.samples[0] != "Film.2023.720p.mkv\nhttps://t.me/c/123/456" {
		t.Fatalf("unexpected refresh samples: %v", audit.samples)
	}
}

func TestAvailabilityQuietHoursAndQuery(t *testing.T) {
	for _, tt := range []struct {
		hour, minute int
		want         bool
	}{
		{0, 59, true}, {1, 0, false}, {6, 59, false}, {7, 0, true}, {23, 59, true},
	} {
		now := time.Date(2026, 11, 28, tt.hour, tt.minute, 0, 0, availabilityIST)
		if got := availabilityHours(now.UTC()); got != tt.want {
			t.Errorf("%v: got %v, want %v", now, got, tt.want)
		}
	}
	parsed := availabilityQuery(42, " HUNGER  Games 2023 Kannada ")
	if parsed.UserID != 42 || parsed.Title != "hunger games" || parsed.Year != "2023" || parsed.Audio != "kan" {
		t.Fatalf("unexpected subscription: %+v", parsed)
	}
}
