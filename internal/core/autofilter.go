package core

import (
	"context"
	"fmt"
 "strings"
	"strconv"
	"time"

	"github.com/Jisin0/autofilterbot/internal/autofilter"
	"github.com/Jisin0/autofilterbot/internal/button"
	"github.com/Jisin0/autofilterbot/internal/format"
	"github.com/Jisin0/autofilterbot/internal/functions"
	"github.com/Jisin0/autofilterbot/pkg/conversation"
	"github.com/Jisin0/autofilterbot/pkg/callbackdata"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"go.uber.org/zap"
)

// PrivateAutofilter provides the same search experience in direct messages as in groups.
// Replies to active admin setup conversations must not be treated as filenames.
func PrivateAutofilter(bot *gotgbot.Bot, ctx *ext.Context) error {
	if ctx.Message == nil || ctx.Message.From == nil || conversation.ActiveListeners.HasMatch(ctx.Message) {
		return nil
	}
	return Autofilter(bot, ctx)
}

func Autofilter(bot *gotgbot.Bot, ctx *ext.Context) error {
	msg, err := _autofilter(bot, ctx)
	if err != nil {
		_app.Log.Warn("autofilter error", zap.Error(err))
	}

	if msg != nil && _app.Config.GetAutodeleteTime() != 0 {
		err := _app.AutoDelete.SaveMessage(msg, time.Minute*time.Duration(_app.Config.AutodeleteTime))
		if err != nil {
			_app.Log.Warn("autofilter: save autodelete failed", zap.Error(err))
		}
	}

	return nil
}

// autofilter runs the autofilter task and returns the sent message.
func _autofilter(bot *gotgbot.Bot, ctx *ext.Context) (*gotgbot.Message, error) {
	var (
		query        string
		inputMessage gotgbot.MaybeInaccessibleMessage
		fromUser     *gotgbot.User
	)

	switch {
	case ctx.CallbackQuery != nil:
		c := ctx.CallbackQuery

		// callback data structure: af:<query>_<user_id>
		callbackData := callbackdata.FromString(c.Data)
		if len(callbackData.Args) < 2 {
			_, err := c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{
				Text:      "Malformed Query: Not Enough Arguments",
				ShowAlert: true,
			})

			_app.Log.Warn("autofilter: bad callback data", zap.Strings("args", callbackData.Args))

			return nil, err
		}

		userId, err := strconv.ParseInt(callbackData.Args[1], 10, 64)
		if err != nil {
			_, err := c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{
				Text:      "Sorry An Error occurred :{",
				ShowAlert: true,
			})
			_app.Log.Warn("autofilter: parse user id failed", zap.Error(err))

			return nil, err
		}

		if c.From.Id != userId {
			_, err := c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{
				Text:      "You Can't Use This Button!", //TODO: customize
				ShowAlert: true,
			})

			return nil, err
		}

		inputMessage = c.Message
		if m, ok := c.Message.(*gotgbot.Message); ok {
			inputMessage = m.ReplyToMessage
		}

		query = callbackData.Args[0]
		fromUser = &c.From
	case ctx.Message != nil:
		m := ctx.Message

		text := m.Text
		if text == "" {
			return nil, nil
		}

		if autofilter.IsBadQueryWithLimit(text, m.Entities, func() int { if m.Chat.Type == "private" { return 120 }; return 30 }()) {
			_app.Log.Debug("autofilter: bad query", zap.String("text", text), zap.Any("entities", m.Entities))
			return nil, nil
		}

		text = autofilter.Sanitize(text)

		inputMessage = m
		query = text
		fromUser = m.From
	default:
		_app.Log.Warn("autofilter: unsupported update type", zap.Int64("update_id", ctx.UpdateId))
		return nil, nil
	}

	if ctx.Message != nil {
        if !CheckSearchAccess(bot,ctx) {return nil,nil}
        _ = _app.DB.LogSearch(query)
    }
	cursor, err := _app.DB.SearchFiles(query)
	if err != nil {
		_app.Log.Warn("autofilter: search files failed", zap.Error(err))
		return bot.SendMessage(inputMessage.GetChat().Id, "<i>I'm Having Some Database Issues Right Now 😓\nPlease Try Again Later!</i>", &gotgbot.SendMessageOpts{
			ReplyParameters: &gotgbot.ReplyParameters{
				MessageId: inputMessage.GetMessageId(),
			},
			ParseMode: gotgbot.ParseModeHTML,
		})
	}

	files, err := autofilter.FilesFromCursor(context.Background(), cursor, _app.Config)
	if err != nil {
		_app.Log.Warn("autofilter: files from cursor failed", zap.Error(err))
		return bot.SendMessage(inputMessage.GetChat().Id, "<i>Processing Results Failed 🤖</i>", &gotgbot.SendMessageOpts{
			ReplyParameters: &gotgbot.ReplyParameters{
				MessageId: inputMessage.GetMessageId(),
			},
			ParseMode: gotgbot.ParseModeHTML,
		})
	}

	if len(files) == 0 {
        suggestions, suggestErr := SuggestTitles(query)
        if suggestErr == nil && len(suggestions)>0 {
            var hints []string
            var keyboard [][]gotgbot.InlineKeyboardButton
            for _,name:=range suggestions {
                hints=append(hints,"• "+name)
                keyboard=append(keyboard,[]gotgbot.InlineKeyboardButton{{Text:"📋 "+name,CopyText:&gotgbot.CopyTextButton{Text:name}}})
            }
            text:="🔎 No exact match. Did you mean?\n"+strings.Join(hints,"\n")+"\nTap to copy and search again."
            return bot.SendMessage(inputMessage.GetChat().Id,text,&gotgbot.SendMessageOpts{
                ReplyParameters:&gotgbot.ReplyParameters{MessageId:inputMessage.GetMessageId()},
                ReplyMarkup:gotgbot.InlineKeyboardMarkup{InlineKeyboard:keyboard},
            })
        }
		vals := _app.BasicMessageValues(ctx, map[string]any{"query": query})
		return bot.SendMessage(inputMessage.GetChat().Id, format.KeyValueFormat(_app.Config.GetNoResultText(), vals), &gotgbot.SendMessageOpts{
			ReplyParameters: &gotgbot.ReplyParameters{
				MessageId: inputMessage.GetMessageId(),
			},
			ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
				{{Text: "Sᴇᴀʀᴄʜ Oɴ Gᴏᴏɢʟᴇ 🔎", Url: fmt.Sprintf("https://google.com/?q=%s", query)}},
				{{Text: "Cᴏᴘʏ", CopyText: &gotgbot.CopyTextButton{Text: query}}, button.Close(fromUser.Id)},
			}},
			ParseMode: gotgbot.ParseModeHTML,
		})
	}

	var warn string
	if _app.Config.GetAutodeleteTime() != 0 {
		warn = fmt.Sprintf("<blockquote><b>⚠️ 𝖳𝗁𝗂𝗌 𝖬𝖾𝗌𝗌𝖺𝗀𝖾 𝖶𝗂𝗅𝗅 𝖡𝖾 𝖠𝗎𝗍𝗈𝗆𝖺𝗍𝗂𝖼𝖺𝗅𝗅𝗒 𝖣𝖾𝗅𝖾𝗍𝖾𝖽 𝖨𝗇 %d 𝖬𝗂𝗇𝗎𝗍𝖾𝗌</b></blockquote>", _app.Config.AutodeleteTime)
	}

	var (
		buttons  = make([][]gotgbot.InlineKeyboardButton, 0, len(files[0])+2)
		uniqueId = functions.RandString(15)
	)

	buttons = append(buttons, headerRow(uniqueId, 0))
	buttons = append(buttons, filterButton(uniqueId))
	buttons = append(buttons, files[0].Process(inputMessage.GetChat().Id, bot.Username, _app.Config)...)
	buttons = append(buttons, footerRow(uniqueId, 0, len(files)))

	text := format.KeyValueFormat(_app.Config.GetResultTemplate(), _app.BasicMessageValues(ctx, map[string]any{"query": query, "warn": warn}))

	msg, err := bot.SendMessage(inputMessage.GetChat().Id, text, &gotgbot.SendMessageOpts{
		ReplyParameters: &gotgbot.ReplyParameters{
			MessageId: inputMessage.GetMessageId(),
		},
		ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: buttons},
		ParseMode:   gotgbot.ParseModeHTML,
	})
	if err != nil {
		_app.Log.Warn("autofilter: send result failed", zap.Error(err))
	}

	err = _app.Cache.Autofilter.Save(&autofilter.SearchResult{
		UniqueId: uniqueId,
		Query:    query,
		FromUser: fromUser.Id,
		ChatID:   ctx.EffectiveChat.Id,
		Files:    files,
	})
	if err != nil {
		_app.Log.Warn("autfilter: save cache failed", zap.Error(err), zap.String("query", query))
	}

	return msg, nil
}

func headerRow(uniqueId string, pageIndex int) []gotgbot.InlineKeyboardButton {
	return []gotgbot.InlineKeyboardButton{allButton(uniqueId, pageIndex), selectButton(uniqueId, pageIndex)}
}

func allButton(uniqueId string, pageIndex int) gotgbot.InlineKeyboardButton {
	return gotgbot.InlineKeyboardButton{Text: "ᴀʟʟ", CallbackData: fmt.Sprintf("all|%s_%d", uniqueId, pageIndex)}
}

func selectButton(uniqueId string, pageIndex int) gotgbot.InlineKeyboardButton {
	return gotgbot.InlineKeyboardButton{Text: "sᴇʟᴇᴄᴛ", CallbackData: fmt.Sprintf("sel|%s_%d", uniqueId, pageIndex)}
}

func footerRow(uniqueId string, pageIndex, totalPages int) []gotgbot.InlineKeyboardButton {
	btns := make([]gotgbot.InlineKeyboardButton, 0, 3)
	if pageIndex != 0 {
		btns = append(btns, backButton(uniqueId, pageIndex-1))
	}

	btns = append(btns, gotgbot.InlineKeyboardButton{Text: fmt.Sprintf("📑 𝗣𝗔𝗚𝗘 %d/%d", pageIndex+1, totalPages), CallbackData: "ignore"})

	if pageIndex+1 != totalPages {
		btns = append(btns, nextButton(uniqueId, pageIndex+1))
	}

	return btns
}

func backButton(uniqueId string, pageIndex int) gotgbot.InlineKeyboardButton {
	return gotgbot.InlineKeyboardButton{Text: "« ʙᴀᴄᴋ", CallbackData: fmt.Sprintf("navg|%s_%d", uniqueId, pageIndex)}
}

func nextButton(uniqueId string, pageIndex int) gotgbot.InlineKeyboardButton {
	return gotgbot.InlineKeyboardButton{Text: "ɴᴇxᴛ »", CallbackData: fmt.Sprintf("navg|%s_%d", uniqueId, pageIndex)}
}
