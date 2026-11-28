package core

import (
	"strings"
	"testing"

	"github.com/Jisin0/autofilterbot/internal/autofilter"
	"github.com/Jisin0/autofilterbot/internal/config"
)

func TestFranchiseCardsShowSeparateYearsAndScopedDownloads(t *testing.T) {
	t.Setenv("OMDB_API_KEY", "test-key")
	first := mockFile("The Hunger Games 2012 1080p English")
	first.OriginalFileName = "The.Hunger.Games.2012.1080p.English.mkv"
	first.UniqueId = "hunger2012"
	first.Audio = map[string]bool{"eng": true}
	sequel := mockFile("The Hunger Games Catching Fire 2013 1080p Tamil Dubbed")
	sequel.OriginalFileName = "The.Hunger.Games.Catching.Fire.2013.1080p.Tamil.Dubbed.mkv"
	sequel.UniqueId = "hunger2013"
	sequel.Audio = map[string]bool{"tam": true, "hin": true}
	sequel.FileSize = 2254857830
	sub := mockFile("The Hunger Games Catching Fire 2013 English")
	sub.OriginalFileName = "The.Hunger.Games.Catching.Fire.2013.English.srt"
	sub.UniqueId = "subtitle2013"
	cards := buildMovieCards([]autofilter.Files{{first, sequel, sub}})
	if len(cards) != 2 || !showMovieCards(cards) || showMovieCards(cards[:1]) {
		t.Fatalf("unexpected film groups %+v", cards)
	}
	list := movieCardsKeyboard("session", cards, 0)
	if !strings.Contains(list[1][0].Text, "2012") || !strings.Contains(list[1][0].Text, "English") ||
		!strings.Contains(list[2][0].Text, "2013") || !strings.Contains(list[2][0].Text, "Tamil") ||
		list[2][0].CallbackData != "card|session_1_0_film" {
		t.Fatalf("unexpected result buttons %+v", list)
	}

	groups := groupDirectMovieFiles([]autofilter.Files{cards[1].files()}, nil)
	buttons := filmKeyboard("session", 1, 0, cards[1], groups, "ta", 123, "AutofilterMGbot", &config.Config{})
	var foundDownload, foundDetails, foundPoster, foundEnglish, foundBack bool
	for _, row := range buttons {
		for _, button := range row {
			if button.Url != "" {
				link := autofilter.URLData{FileUniqueId: sequel.UniqueId, ChatId: 123}.Encode()
				if !strings.HasSuffix(button.Url, "?start="+link) || !strings.HasPrefix(button.Text, "1080p · Tamil + Hindi · ") {
					t.Fatalf("incorrect file or compact label: %+v", button)
				}
				foundDownload = true
			}
			switch button.CallbackData {
			case "fdetails|hunger2013":
				foundDetails = button.Text == "Details"
			case "card|session_1_0_poster":
				foundPoster = true
			case "card|session_1_0_en":
				foundEnglish = true
			case "cards|session_0":
				foundBack = true
			}
		}
	}
	if !foundDownload || !foundDetails || !foundPoster || !foundEnglish || !foundBack {
		t.Fatalf("missing film controls %+v", buttons)
	}
}

func TestMoviePosterFallbackAndCompactTemplateCompatibility(t *testing.T) {
	t.Setenv("OMDB_API_KEY", "")
	if len(posterButton("session", 0, movieCard{Title: "Hunger Games", Year: "2012"})) != 0 {
		t.Fatal("poster control should be omitted when OMDb is unavailable")
	}
	for _, raw := range []string{"N/A", "http://example.com/poster.jpg", "https://user@example.com/poster.jpg"} {
		if validPosterURL(raw) {
			t.Errorf("unexpected poster URL %q", raw)
		}
	}
	file := mockFile("Hunger.Games.2012.1080p.mkv")
	file.UniqueId = "example"
	result := autofilter.Files{file}.ProcessMovie(123, "AutofilterMGbot", &config.Config{ButtonTemplate: "Original {file_name}"})
	if len(result[0]) != 1 || result[0][0].Text != "Original "+file.FileName {
		t.Fatalf("custom format changed %+v", result)
	}
}
