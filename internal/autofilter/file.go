package autofilter

import (
	"fmt"
	"github.com/Jisin0/autofilterbot/internal/captiontags"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Jisin0/autofilterbot/internal/format"
	"github.com/Jisin0/autofilterbot/internal/functions"
	"github.com/Jisin0/autofilterbot/internal/model"
	"github.com/Jisin0/autofilterbot/pkg/shortener"
	"github.com/PaulSonOfLars/gotgbot/v2"
)

// Only render the verified Fringe abbreviation as a readable title.
// The indexed filename and Telegram file ID remain unchanged.
var fringeShortName = regexp.MustCompile(`(?i)^fri([._ -]+s[0-9]{1,2}[ ._-]*e[0-9]{1,3}(?:[^a-z0-9]|$))`)

func displayFileName(name string) string {
	if fringeShortName.MatchString(name) {
		return "Fringe" + name[3:]
	}
	return name
}

type Files []File

// File wraps model.File with some search result specific data.
type File struct {
	model.File
	Audio map[string]bool `json:"audio,omitempty"`
	// Indicates whether the file was selected from the selection menu.
	IsSelected bool `json:"selected,omitempty"`
}

// Process returns a slice of buttons to be used in message markup.
func (files Files) Process(chatId int64, botUsername string, opts ProcessFilesOptions) [][]gotgbot.InlineKeyboardButton {
	return ProcessFiles(files, chatId, botUsername, opts)
}

type ProcessFilesOptions interface {
	GetButtonTemplate() string
	GetSizeButton() bool
	GetShortener() shortener.Shortener
}

// ProcessFiles changes files into a keboard slice to be used as markup in a message.
func ProcessFiles(files Files, chatId int64, botUsername string, opts ProcessFilesOptions) [][]gotgbot.InlineKeyboardButton {
	return processFiles(files, chatId, botUsername, opts, false)
}

// ProcessMovie uses compact labels within a single movie's file list.
func (files Files) ProcessMovie(chatID int64, username string, opts ProcessFilesOptions) [][]gotgbot.InlineKeyboardButton {
	return processFiles(files, chatID, username, opts, true)
}

var releaseQuality = regexp.MustCompile(`(?i)(?:^|[^a-z0-9])(360p|480p|720p|1080p|2160p|4k)(?:[^a-z0-9]|$)`)

func processFiles(files Files, chatId int64, botUsername string, opts ProcessFilesOptions, compact bool) [][]gotgbot.InlineKeyboardButton {
	var (
		hasShortener = opts.GetShortener().ApiKey != ""
		result       = make([][]gotgbot.InlineKeyboardButton, 0, len(files))
	)

	for _, f := range files {
		url := fmt.Sprintf("https://t.me/%s?start=%s", botUsername, URLData{
			FileUniqueId: f.UniqueId,
			ChatId:       chatId,
			HasShortener: hasShortener,
		}.Encode())
		size := functions.FileSizeToString(f.FileSize)
		displayName := displayFileName(f.FileName)
		var labels []string
		for _, code := range captiontags.AudioCodes {
			if f.Audio[code] {
				labels = append(labels, captiontags.LanguageLabel(code))
			}
		}
		if compact && (opts.GetSizeButton() || opts.GetButtonTemplate() == "📂 {file_size} {file_name}") {
			if label := compactMovieFile(f, labels, size); label != "" {
				result = append(result, []gotgbot.InlineKeyboardButton{
					{Text: label, Url: url},
					{Text: "Details", CallbackData: "fdetails|" + f.UniqueId},
				})
				continue
			}
		}
		if len(labels) > 0 {
			displayName += " • Audio: " + strings.Join(labels, ", ")
		}

		if opts.GetSizeButton() {
			result = append(result, []gotgbot.InlineKeyboardButton{{Text: displayName, CallbackData: "fdetails|" + f.UniqueId}, {Text: size, Url: url}})
		} else {
			text := format.KeyValueFormat(opts.GetButtonTemplate(), map[string]string{
				"file_name": displayName,
				"file_size": size,
			})
			result = append(result, []gotgbot.InlineKeyboardButton{{Text: text, Url: url}})
		}
	}

	return result
}

func compactMovieFile(file File, labels []string, size string) string {
	name := file.OriginalFileName
	if name == "" {
		name = file.FileName
	}
	quality := releaseQuality.FindStringSubmatch(name)
	if ext := filepath.Ext(name); ext != "" && !captiontags.IsVideoUpload(name, false) {
		return ""
	}
	if !captiontags.IsVideoUpload(name, file.FileType == model.FileTypeVideo) && len(quality) == 0 {
		return ""
	}
	var parts []string
	if len(quality) > 0 {
		label := strings.ToLower(quality[1])
		if label == "2160p" || label == "4k" {
			label = "4K"
		}
		parts = append(parts, label)
	}
	if len(labels) > 2 {
		parts = append(parts, fmt.Sprintf("%s + %d", labels[0], len(labels)-1))
	} else if len(labels) > 0 {
		parts = append(parts, strings.Join(labels, " + "))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(append(parts, size), " · ")
}

// SelectMenu returns a keyboard with to select files from.
func (files Files) SelectMenu(uniqueId string, pageIndex int) [][]gotgbot.InlineKeyboardButton {
	keyboard := make([][]gotgbot.InlineKeyboardButton, 0, len(files))

	for _, f := range files {
		keyboard = append(keyboard, []gotgbot.InlineKeyboardButton{{
			Text:         fmt.Sprintf("%s[%s] %s", tick(f.IsSelected), functions.FileSizeToString(f.FileSize), displayFileName(f.FileName)),
			CallbackData: fmt.Sprintf("sel|%s_%d_%s", uniqueId, pageIndex, f.UniqueId),
		}})
	}

	return keyboard
}

// tick returns a tick symbol if val is true.
func tick(val bool) string {
	if val {
		return "✅ "
	}

	return ""
}
