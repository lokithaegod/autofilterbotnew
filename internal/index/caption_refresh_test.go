package index

import (
	"github.com/amarnathcjd/gogram/telegram"
	"testing"
)

func TestScanCaptionMessagesOnlyMarksCaptionedVideos(t *testing.T) {
	cases := []struct {
		name, caption, filename string
		want                    int
	}{
		{"movie caption", "Malayalam Sub", "A.Quiet.Place.2018.1080p.mkv", 1},
		{"another form", "✅ Malayalam Subs", "Cast.Away.2000.720p.mp4", 1},
		{"no caption", "English subtitles", "Movie.2000.1080p.mkv", 0},
		{"negative caption", "No Malayalam subtitles available", "Movie.2000.mkv", 0},
		{"subtitle isn't video", "Malayalam Sub", "Movie.2000.srt", 0},
		{"zip isn't video", "Malayalam Sub", "Movie.2000.മലയാളം.zip", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			history := &telegram.MessagesMessagesObj{Messages: []telegram.Message{
				&telegram.MessageObj{ID: 17, Message: tc.caption, Media: &telegram.MessageMediaDocument{
					Document: &telegram.DocumentObj{Size: 100, Attributes: []telegram.DocumentAttribute{
						&telegram.DocumentAttributeFilename{FileName: tc.filename},
					}},
				}},
			}}
			found := 0
			refreshed := 0
			result := CaptionScanResult{}
			err := scanCaptionMessages(history, -1001234567890, func(id, name string, size int64, enabled bool) (int, error) {
				found++
				if id == "" || name != tc.filename || size != 100 || !enabled {
					t.Errorf("unexpected tag data: %q %q %d %v", id, name, size, enabled)
				}
				return 2, nil
			}, func(id, name string, size int64, caption string) (int, error) {
				refreshed++
				if id == "" || name != tc.filename || size != 100 || caption != tc.caption {
					t.Errorf("unexpected caption data: %q %q %d %q", id, name, size, caption)
				}
				return 1, nil
			}, &result)
			if err != nil {
				t.Fatal(err)
			}
			wantRefresh := 1
			if tc.name == "subtitle isn't video" || tc.name == "zip isn't video" {
				wantRefresh = 0
			}
			if refreshed != wantRefresh || result.CaptionsRefreshed != wantRefresh {
				t.Fatalf("refreshed %d and counted %d, want %d", refreshed, result.CaptionsRefreshed, wantRefresh)
			}
			if found != tc.want || result.Captioned != tc.want || result.Scanned != 1 {
				t.Fatalf("found %d scanned %d captioned %d, want %d", found, result.Scanned, result.Captioned, tc.want)
			}
		})
	}
}
