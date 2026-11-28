package captiontags

import (
	"maps"
	"testing"
)

func TestAudioLanguages(t *testing.T) {
	for _, tc := range []struct {
		name, caption string
		want          []string
	}{
		{"Movie.2023.mkv", "Audio: [Tamil + Telugu] | Subs: [Malayalam + English]", []string{"tam", "tel"}},
		{"Movie.2023.mkv", "Audio Language: Hindi & Kannada\nEnglish Sub", []string{"hin", "kan"}},
		{"Movie.2023.mkv", "தமிழ் டப் | Malayalam Sub", []string{"tam"}},
		{"Movie.2023.mkv", "Subtitles: Tamil, Hindi and Malayalam", nil},
		{"Movie.2023.mkv", "English Audio | English Sub", []string{"eng"}},
		{"Movie.2023.Tamil.mkv", "Tamil Sub | English Audio", []string{"eng"}},
		{"Movie.2023.Tamil.Audio.mkv", "Tamil subtitles included", []string{"tam"}},
		{"Movie.2023.Tamil.mkv", "Tamil Audio unavailable", nil},
		{"Movie.2023.Tamil.Subs.English.mkv", "", nil},
		{"The.English.Patient.1996.1080p.mkv", "", nil},
		{"Movie.2023.Hindi.Telugu.1080p.mkv", "", []string{"hin", "tel"}},
		{"Movie.2023.mkv", "Audio: Malayalam + English; Hindi Dubbed", []string{"mal", "eng", "hin"}},
		{"Movie.2023.mkv", "Sub Tamil | Subtitle: Kannada", nil},
	} {
		t.Run(tc.name+"/"+tc.caption, func(t *testing.T) {
			want := make(map[string]bool)
			for _, code := range tc.want {
				want[code] = true
			}
			if got := AudioLanguages(tc.name, tc.caption); !maps.Equal(got, want) {
				t.Fatalf("audio languages = %v, want %v", got, want)
			}
		})
	}
}
