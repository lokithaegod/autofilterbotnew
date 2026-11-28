package core

import (
	"github.com/Jisin0/autofilterbot/internal/captiontags"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/Jisin0/autofilterbot/internal/format"
	"github.com/Jisin0/autofilterbot/internal/functions"
	"github.com/Jisin0/autofilterbot/pkg/callbackdata"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"go.uber.org/zap"
)

const (
	fiveHoursInSeconds int64 = 5 * 60 * 60
)

// Close handles callback events from close buttons.
func Close(bot *gotgbot.Bot, ctx *ext.Context) error {
	c := ctx.CallbackQuery

	data := callbackdata.FromString(c.Data)
	if len(data.Args) == 0 { // no authorized users, anyone can use
		c.Message.Delete(bot, nil)
		return nil
	}

	for _, s := range data.Args {
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			_app.Log.Warn("close: parse id failed", zap.Error(err), zap.String("id", s))
			continue
		}

		if id == c.From.Id {
			c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "Message Has Been Deleted 🗑️"})
			c.Message.Delete(bot, nil)
			return nil
		}
	}

	_, err := c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{
		Text:      "You Can't Use This Button",
		ShowAlert: true,
	})
	if err != nil {
		_app.Log.Warn("close: answer query failed", zap.Error(err))
	}

	return nil
}

// Ignore handles the ignore callback.
func Ignore(bot *gotgbot.Bot, ctx *ext.Context) error {
	ctx.CallbackQuery.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{CacheTime: fiveHoursInSeconds})
	return nil
}

// FileDetails handles the fdetails callback query to print details about a file.
func FileDetails(bot *gotgbot.Bot, ctx *ext.Context) error {
	c := ctx.CallbackQuery

	data := callbackdata.FromString(c.Data)
	if len(data.Args) == 0 {
		c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "Error: No Callback Args"})
		return nil
	}

	fileUniqueId := data.Args[0]

	f, err := _app.DB.GetFile(fileUniqueId)
	if err != nil {
		c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "404: File Not Found", ShowAlert: true})
		_app.Log.Debug("fdetails: get file failed", zap.Error(err), zap.String("unique_id", fileUniqueId))
		return nil
	}

	name := sourceFileName(*f)
	text := format.KeyValueFormat(_app.Config.GetFileDetailsTemplate(), map[string]string{
		"file_name": name,
		"file_size": functions.FileSizeToString(f.FileSize),
		"file_type": f.FileType,
		"date":      functions.FormatUnixTimestamp(f.Time),
	})

	if !strings.Contains(text, name) {
		text = name + "\n\n" + text
	}
	legacyTamil := false
	if f.Caption == nil {
		tags, _ := _app.DB.FindTamilDubCaptionTags([]string{f.UniqueId})
		legacyTamil = tags[f.UniqueId]
	}
	audio := detectedAudio(*f, legacyTamil)
	var languages []string
	for _, code := range captiontags.AudioCodes {
		if audio[code] {
			languages = append(languages, captiontags.LanguageLabel(code))
		}
	}
	if len(languages) > 0 {
		text += "\nAudio: " + strings.Join(languages, ", ")
	}
	if utf8.RuneCountInString(text) > 200 && c.Message != nil {
		_, _ = c.Answer(bot, nil)
		_, err = bot.SendMessage(c.Message.GetChat().Id, clippedLabel(text, 4096), nil)
		return err
	}

	_, err = c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: text, ShowAlert: true, CacheTime: fiveHoursInSeconds})
	if err != nil {
		_app.Log.Warn("fdetails: answer query failed", zap.Error(err))
	}

	return nil
}
