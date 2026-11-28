package captiontags

import (
	"regexp"
	"strings"
)

var languageAliases = map[string]string{
	"tamil": "tam", "tam": "tam", "தமிழ்": "tam",
	"hindi": "hin", "hin": "hin", "हिंदी": "hin", "हिन्दी": "hin",
	"malayalam": "mal", "malyalam": "mal", "mal": "mal", "മലയാളം": "mal",
	"telugu": "tel", "tel": "tel", "తెలుగు": "tel",
	"kannada": "kan", "kan": "kan", "ಕನ್ನಡ": "kan",
	"english": "eng", "eng": "eng",
}

func LanguageCode(word string) string { return languageAliases[strings.ToLower(word)] }

var (
	languageWords    = regexp.MustCompile(`[\pL\pM\pN]+`)
	subtitleHeading  = regexp.MustCompile(`(?i)\b(?:subs?|subbed|subtitles?|esubs?|msubs?|srt|cc)\b\s*[:=–—-]`)
	audioHeading     = regexp.MustCompile(`(?i)\b(?:audio(?:\s+(?:tracks?|languages?))?|languages?|dubbed\s+in)\b\s*[:=–—-]`)
	negativeHeading  = regexp.MustCompile(`(?i)\b(?:no|without)\s+audio\s*[:=–—-]`)
	clauseText       = regexp.MustCompile(`[^\r\n;|•]+`)
	releaseTokens    = regexp.MustCompile(`(?i)\b(?:mp3|aac|ac3|eac3|dts|flac|kbps|hd|4k|480p|720p|1080p|2160p)\b`)
	releaseYearToken = regexp.MustCompile(`(?:^|[^0-9])(?:19|20)[0-9]{2}(?:$|[^a-zA-Z0-9])`)
)

func subtitleWord(word string) bool {
	switch word {
	case "sub", "subs", "subtitle", "subtitles", "subbed", "esub", "esubs", "msub", "msubs", "srt", "cc":
		return true
	}
	return false
}

func audioWord(word string) bool {
	switch word {
	case "audio", "dub", "dubbed", "dubbing", "டப்":
		return true
	}
	return false
}

// AudioLanguages reads release labels and caption declarations. Subtitle
// headings and markers take precedence within their own clause.
func AudioLanguages(filename, caption string) map[string]bool {
	found := make(map[string]bool)
	excluded := make(map[string]bool)
	nameLabels := make(map[string]bool)
	subtitles := make(map[string]bool)
	readAudioLabels(filename, true, found, nameLabels, subtitles, excluded)
	readAudioLabels(caption, false, found, nil, subtitles, excluded)
	for code := range nameLabels {
		if !subtitles[code] {
			found[code] = true
		}
	}
	for code := range excluded {
		delete(found, code)
	}
	return found
}

func readAudioLabels(text string, filename bool, found, nameLabels, subtitles, excluded map[string]bool) {
	if filename {
		if year := releaseYearToken.FindStringIndex(text); year != nil {
			text = text[year[1]:]
		}
	}
	text = subtitleHeading.ReplaceAllString(strings.ToLower(text), "; subtitles:")
	text = negativeHeading.ReplaceAllString(text, "; noaudio:")
	text = audioHeading.ReplaceAllString(text, "; audio:")

	previousEnd := 0
	previousSubtitles := false
	for _, bounds := range clauseText.FindAllStringIndex(text, -1) {
		sameLineList := strings.TrimSpace(text[previousEnd:bounds[0]]) == "|"
		clause := text[bounds[0]:bounds[1]]
		previousEnd = bounds[1]
		words := languageWords.FindAllString(clause, -1)
		if len(words) == 0 {
			previousSubtitles = false
			continue
		}
		subtitleList := subtitleWord(words[0])
		audioList := audioWord(words[0]) || (len(words) > 1 && audioWord(words[1]) && words[0] == "dual")
		// A trailing role can describe an entire standalone list.
		last := words[len(words)-1]
		list := words
		if subtitleWord(last) || audioWord(last) {
			list = words[:len(words)-1]
		}
		onlyLanguages := len(list) > 0
		for _, word := range list {
			if LanguageCode(word) == "" && word != "and" {
				onlyLanguages = false
			}
		}
		subtitleList = subtitleList || (onlyLanguages && subtitleWord(last)) ||
			(sameLineList && previousSubtitles && onlyLanguages && !audioWord(last))
		previousSubtitles = subtitleList
		negativeList := words[0] == "noaudio"

		for i, word := range words {
			code := LanguageCode(word)
			if code == "" {
				continue
			}
			var prev, next, after string
			if i > 0 {
				prev = words[i-1]
			}
			if i+1 < len(words) {
				next = words[i+1]
			}
			if i+2 < len(words) {
				after = words[i+2]
			}
			if subtitleList || subtitleWord(next) || (subtitleWord(prev) && !audioWord(next) && !audioList) {
				subtitles[code] = true
				continue
			}
			if negativeList || prev == "no" || prev == "without" || prev == "not" || next == "unavailable" ||
				(next == "not" && after == "available") ||
				(audioWord(next) && (after == "not" || after == "unavailable")) {
				excluded[code] = true
				continue
			}
			if audioList || audioWord(prev) || audioWord(next) || (!filename && (onlyLanguages || releaseTokens.MatchString(clause))) {
				found[code] = true
			} else if filename {
				nameLabels[code] = true
			}
		}
	}
}
