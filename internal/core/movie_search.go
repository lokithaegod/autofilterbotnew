package core

import (
	"context"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Jisin0/autofilterbot/internal/autofilter"
	"github.com/Jisin0/autofilterbot/internal/captiontags"
	"github.com/Jisin0/autofilterbot/internal/database"
	"github.com/Jisin0/autofilterbot/internal/model"
)

var releaseYear = regexp.MustCompile(`^(?:19|20)[0-9]{2}$`)

type movieSearch struct {
	title string
	year  string
	audio string
}

func parseMovieSearch(query string) movieSearch {
	result := movieSearch{title: query}
	words := strings.Fields(query)
	for len(words) > 1 {
		last := strings.ToLower(words[len(words)-1])
		if result.year == "" && releaseYear.MatchString(last) {
			result.year = last
			words = words[:len(words)-1]
			continue
		}
		end := len(words) - 1
		if end > 1 && (last == "dubbed" || last == "dub" || last == "audio") {
			end--
		}
		if code := captiontags.LanguageCode(words[end]); result.audio == "" && code != "" {
			result.audio = code
			words = words[:end]
			continue
		}
		break
	}
	if result.year != "" || result.audio != "" {
		result.title = strings.Join(words, " ")
	}
	return result
}

func sourceFileName(file model.File) string {
	if file.OriginalFileName != "" {
		return file.OriginalFileName
	}
	return file.FileName
}

func canHaveMovieAudio(file model.File) bool {
	name := sourceFileName(file)
	if isMovieSubtitleDocument(name) || file.FileType == model.FileTypeVoice {
		return false
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".pdf", ".epub", ".doc", ".docx", ".ppt", ".pptx", ".txt", ".jpg", ".jpeg", ".png", ".gif", ".webp", ".rar", ".7z":
		return false
	}
	return true
}

func detectedAudio(file model.File, legacyTamil bool) map[string]bool {
	name := sourceFileName(file)
	if !canHaveMovieAudio(file) {
		return nil
	}
	caption := ""
	if file.Caption != nil {
		caption = *file.Caption
	}
	audio := captiontags.AudioLanguages(name, caption)
	if file.Caption == nil && legacyTamil {
		audio["tam"] = true
	}
	return audio
}

func matchesAudio(file model.File, language string, legacyTamil bool) bool {
	return detectedAudio(file, legacyTamil)[language]
}

// Read candidates in batches so legacy Tamil tags can be loaded together and
// the existing display limit applies to matching files.
func audioSearchFiles(ctx context.Context, cursor database.Cursor, options fullSeriesSearchOptions, language string, findTamil func([]string) (map[string]bool, error)) ([]autofilter.Files, error) {
	return collectSearchFiles(ctx, cursor, options.GetMaxPages()*options.GetMaxPerPage(), options.GetMaxPerPage(), language, findTamil, false)
}

func collectSearchFiles(ctx context.Context, cursor database.Cursor, maxFiles, perPage int, language string, findTamil func([]string) (map[string]bool, error), mediaOnly bool) ([]autofilter.Files, error) {
	const batchSize = 100
	var pages []autofilter.Files
	var page autofilter.Files
	for count := 0; count < maxFiles; {
		batch := make([]model.File, 0, batchSize)
		var ids []string
		for len(batch) < batchSize && cursor.Next(ctx) {
			var file model.File
			if err := cursor.Decode(&file); err != nil {
				return nil, err
			}
			batch = append(batch, file)
			if file.Caption == nil {
				ids = append(ids, file.UniqueId)
			}
		}
		var legacy map[string]bool
		if len(ids) > 0 {
			var err error
			legacy, err = findTamil(ids)
			if err != nil && language == "tam" {
				return nil, err
			}
		}
		for _, file := range batch {
			if mediaOnly && !canHaveMovieAudio(file) {
				continue
			}
			audio := detectedAudio(file, legacy[file.UniqueId])
			if language != "" && language != "all" && !audio[language] {
				continue
			}
			page = append(page, autofilter.File{File: file, Audio: audio})
			count++
			if len(page) == perPage {
				pages = append(pages, page)
				page = nil
			}
			if count == maxFiles {
				break
			}
		}
		if len(batch) < batchSize {
			break
		}
	}
	if len(page) > 0 {
		pages = append(pages, page)
	}
	return pages, ctx.Err()
}
