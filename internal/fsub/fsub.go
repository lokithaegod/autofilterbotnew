package fsub

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Jisin0/autofilterbot/internal/button"
	"github.com/Jisin0/autofilterbot/internal/config"
		"github.com/Jisin0/autofilterbot/internal/database/mongo"
	"github.com/Jisin0/autofilterbot/internal/format"
	"github.com/Jisin0/autofilterbot/internal/model"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"go.uber.org/zap"

	pkgerrors "github.com/pkg/errors"
)

// FsubChannel is a single force sub channel.
type FsubChannel struct {
	model.Channel
}

// IsJoined reports wether the user is a member of the channel.
//
// - bot: bot client to make api requests.
// - userId: id of user to check for.
//
// NOTE: If some error other than "user not found" occurs during api call function returns true and error.
func IsMember(bot *gotgbot.Bot, chatId, userId int64) (bool, error) {
	member, err := bot.GetChatMember(chatId, userId, nil)
	if err != nil {
		if strings.Contains(err.Error(), "user not found") {
			return false, nil
		}

		// if other error returns true with error
		return false, fmt.Errorf("fsub: failed to get chat member: %w", err)
	}

	switch m := member.(type) {
	case gotgbot.ChatMemberLeft, gotgbot.ChatMemberBanned:
		return false, nil
	case gotgbot.ChatMemberRestricted:
		return m.IsMember, nil
	default:
		return true, nil
	}
}

// GetNotMemberOrRequest requires current verified membership in every channel.
// Historical join-request records are deliberately ignored: they do not prove
// the request is still pending and must never bypass ForceSub.
func GetNotMemberOrRequest(bot *gotgbot.Bot, db *mongo.Client, f []model.Channel, userId int64) ([]model.Channel, error) {
 _ = db // Kept in the signature to avoid changing existing integrations.
 var errs []error
 missing := make([]model.Channel, 0, len(f))
 for _, channel := range f {
  metricsMembershipChecks.Add(1)
  joined, err := IsMember(bot, channel.ID, userId)
  if err != nil {
   metricsMembershipErrors.Add(1)
   errs = append(errs, fmt.Errorf("channel %d: %w", channel.ID, err))
   continue
  }
  if !joined { missing = append(missing, channel) }
 }
 return missing, errors.Join(errs...)
}

type appPreview interface {
	GetDB() *mongo.Client
	GetConfig() *config.Config
	GetLog() *zap.Logger
	BasicMessageValues(ctx *ext.Context, extraValues ...map[string]any) map[string]string
}

// CheckFsub checks wether the user has joined or sent a join request to all force subscribe channels.
// The user is sent a message directing them to join the required channels.
//
// Returns boolean indicating whether the user has joined all channels and error indicating an API or Db request error.
//
// NOTE: true will be returned incase of an error.
func CheckFsub(app appPreview, bot *gotgbot.Bot, ctx *ext.Context) (bool, error) {
 metricsChecks.Add(1)
	var (
		userID int64
		chatID int64
	)

	switch {
	case ctx.Message != nil:
		userID = ctx.Message.From.Id
		chatID = ctx.Message.Chat.Id
	case ctx.CallbackQuery != nil:
		userID = ctx.CallbackQuery.From.Id
		chatID = ctx.CallbackQuery.From.Id
	case ctx.InlineQuery != nil:
		userID = ctx.InlineQuery.From.Id
		chatID = ctx.InlineQuery.From.Id
	}

	notJoined, err := GetNotMemberOrRequest(bot, app.GetDB(), app.GetConfig().GetFsubChannels(), userID)
	if err != nil {
  metricsErrors.Add(1)
		// Never send files if Telegram membership or the database cannot be checked.
		if ctx.InlineQuery != nil {
			_, _ = ctx.InlineQuery.Answer(bot, []gotgbot.InlineQueryResult{}, &gotgbot.AnswerInlineQueryOpts{CacheTime: 0})
		} else {
			_, _ = bot.SendMessage(chatID, "⚠️ Unable to verify channel membership right now. Please try again later.", nil)
		}
		return false, pkgerrors.Wrap(err, "fsub: ")
	}

	if len(notJoined) == 0 {
  metricsPassed.Add(1)
		return true, nil
	}

	metricsBlocked.Add(1)
 var btns [][]gotgbot.InlineKeyboardButton

	switch len(notJoined) {
	case 1:
		btns = [][]gotgbot.InlineKeyboardButton{{{Text: "ᴊᴏɪɴ ᴍʏ ᴄʜᴀɴɴᴇʟ", Url: notJoined[0].InviteLink}}}
	case 2:
		btns = [][]gotgbot.InlineKeyboardButton{
			{{Text: "ᴊᴏɪɴ ғɪʀsᴛ ᴄʜᴀɴɴᴇʟ", Url: notJoined[0].InviteLink}},
			{{Text: "ᴊᴏɪɴ sᴇᴄᴏɴᴅ ᴄʜᴀɴɴᴇʟ", Url: notJoined[1].InviteLink}},
		}
	default:
		btns = make([][]gotgbot.InlineKeyboardButton, 0, len(notJoined)+1)
		for i, c := range notJoined {
			btns = append(btns, []gotgbot.InlineKeyboardButton{{Text: fmt.Sprintf("ᴊᴏɪɴ ᴄʜᴀɴɴᴇʟ %d", i+1), Url: c.InviteLink}})
		}
	}

	text := format.KeyValueFormat(app.GetConfig().GetFsubText(), app.BasicMessageValues(ctx))

	retryButton := gotgbot.InlineKeyboardButton{Text: "ʀᴇᴛʀʏ 🔃"}

	switch {
	case ctx.Message != nil:
  // Retain the original deep-link payload if one was supplied. Never index
  // Args()[1] on /start without an argument.
  args := ctx.Args()
  if len(args)>1 && args[1]!="" {
   retryButton.Url=fmt.Sprintf("https://t.me/%s?start=%s",bot.Username,args[1])
  } else {
   retryButton.Url=fmt.Sprintf("https://t.me/%s?start",bot.Username)
  }
	case ctx.CallbackQuery != nil:
		retryButton.CallbackData = ctx.CallbackQuery.Data
	case ctx.InlineQuery != nil:
		retryButton.SwitchInlineQueryCurrentChat = &ctx.InlineQuery.Query
	}

	metricsJoinPrompts.Add(1)
 btns = append(btns,
		[]gotgbot.InlineKeyboardButton{
			button.Close(userID),
			retryButton,
		})

	if ctx.InlineQuery != nil {
		_, err := ctx.InlineQuery.Answer(bot, []gotgbot.InlineQueryResult{gotgbot.InlineQueryResultArticle{
			Id:    "retry",
			Title: "Join My Channels First 👇",
			InputMessageContent: gotgbot.InputTextMessageContent{
				MessageText: text,
				ParseMode:   gotgbot.ParseModeHTML,
			},
			ReplyMarkup: &gotgbot.InlineKeyboardMarkup{InlineKeyboard: btns},
		}},
			&gotgbot.AnswerInlineQueryOpts{
				CacheTime: 5,
			})

		return false, pkgerrors.Wrap(err, "fsub: failed to answer inline query: ")
	}

	_, err = bot.SendMessage(chatID,
		text,
		&gotgbot.SendMessageOpts{
			ParseMode:   gotgbot.ParseModeHTML,
			ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: btns},
		},
	)

	return false, pkgerrors.Wrap(err, "fsub: send fsub message failed: ")
}
