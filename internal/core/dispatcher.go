package core

import (
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/Jisin0/autofilterbot/pkg/conversation"
	"github.com/Jisin0/autofilterbot/pkg/env"
	exthandlers "github.com/Jisin0/autofilterbot/pkg/filters"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/callbackquery"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/message"
	"go.uber.org/zap"
)

const (
	autofilterHandlerGroup = iota + 1
	commandHandlerGroup
	callbackQueryGroup
	miscHandlerGroup
	middleWareGroup
	joinRequestGroup
)

// SetupDispatcher creates a new empty dispatcher with error and panic recovery setup.
func SetupDispatcher(log *zap.Logger) *ext.Dispatcher {
	d := ext.NewDispatcher(&ext.DispatcherOpts{
		// If an error is returned by a handler, log it and continue going.
		Error: func(b *gotgbot.Bot, ctx *ext.Context, err error) ext.DispatcherAction {
			logFields := []zap.Field{zap.Error(err)}
			logFields = addLogFieldsFromContext(ctx, logFields)

			log.Error("error occurred while handling update", logFields...)

			return ext.DispatcherActionNoop
		},

		UnhandledErrFunc: func(err error) {
			log.Debug("dispatcher: unhandled error", zap.Error(err))
		},

		Panic: func(b *gotgbot.Bot, ctx *ext.Context, r interface{}) {
			logFields := []zap.Field{zap.String("panic", fmt.Sprintf("%v\n%s", r, cleanedStack()))}
			logFields = addLogFieldsFromContext(ctx, logFields)

			log.Error("panic recovered", logFields...)
		},
	})

	d.AddHandlerToGroup(handlers.NewMessage(message.Supergroup, Autofilter), autofilterHandlerGroup)
	d.AddHandlerToGroup(handlers.NewMessage(message.Private, PrivateLastFile), autofilterHandlerGroup)

	d.AddHandlerToGroup(handlers.NewCommand("start", StartCommand), commandHandlerGroup)
	d.AddHandlerToGroup(exthandlers.NewCommands([]string{"about", "help", "privacy"}, StaticCommands), commandHandlerGroup)
	d.AddHandlerToGroup(handlers.NewCommand("delete", DeleteFile).SetAllowChannel(true), commandHandlerGroup)
	d.AddHandlerToGroup(exthandlers.NewCommands([]string{"deleteall", "delall"}, DeleteAllFiles), commandHandlerGroup)
	d.AddHandlerToGroup(exthandlers.NewCommands([]string{"settings", "configs", "configpanel"}, Settings), commandHandlerGroup)
	d.AddHandlerToGroup(handlers.NewCommand("logs", Logs), commandHandlerGroup)
	d.AddHandlerToGroup(handlers.NewCommand("stats", Stats), commandHandlerGroup)
	d.AddHandlerToGroup(handlers.NewCommand("dashboard", AdminDashboard), commandHandlerGroup)
	d.AddHandlerToGroup(handlers.NewCommand("admin", AdminMenu), commandHandlerGroup)
	d.AddHandlerToGroup(handlers.NewCommand("alive", Alive), commandHandlerGroup)
	d.AddHandlerToGroup(handlers.NewCommand("health", HealthCommand), commandHandlerGroup)
	d.AddHandlerToGroup(handlers.NewCommand("find", AdvancedFind), commandHandlerGroup)
	d.AddHandlerToGroup(handlers.NewCommand("browse", BrowseFiles), commandHandlerGroup)
	d.AddHandlerToGroup(handlers.NewCommand("movie", MovieInfo), commandHandlerGroup)
	d.AddHandlerToGroup(handlers.NewCommand("banuser", SetBan), commandHandlerGroup)
	d.AddHandlerToGroup(handlers.NewCommand("unbanuser", SetBan), commandHandlerGroup)
	d.AddHandlerToGroup(handlers.NewCommand("batch", NewBatch), commandHandlerGroup)
	d.AddHandlerToGroup(handlers.NewCommand("genlink", GenLink), commandHandlerGroup)
	d.AddHandlerToGroup(handlers.NewCommand("broadcast", Broadcast), commandHandlerGroup)
	d.AddHandlerToGroup(handlers.NewCommand("index", CmdIndex), commandHandlerGroup)
	d.AddHandlerToGroup(handlers.NewCommand("updateindex", AutoIndexLast), commandHandlerGroup)
	d.AddHandlerToGroup(handlers.NewCommand("setwelcome", SetWelcome), commandHandlerGroup)
	d.AddHandlerToGroup(handlers.NewCommand("resetwelcome", ResetWelcome), commandHandlerGroup)
	d.AddHandlerToGroup(handlers.NewCommand("setwelcomebutton", SetWelcomeButton), commandHandlerGroup)
	d.AddHandlerToGroup(handlers.NewCommand("resetwelcomebuttons", ResetWelcomeButtons), commandHandlerGroup)

	d.AddHandlerToGroup(handlers.NewCallback(callbackquery.Prefix("cmd"), StaticCommands), callbackQueryGroup)
	d.AddHandlerToGroup(handlers.NewCallback(callbackquery.Prefix("close"), Close), callbackQueryGroup)
	d.AddHandlerToGroup(handlers.NewCallback(callbackquery.Prefix("navg"), Navigate), callbackQueryGroup)
	d.AddHandlerToGroup(handlers.NewCallback(callbackquery.Prefix("fdetails"), FileDetails), callbackQueryGroup)
	d.AddHandlerToGroup(handlers.NewCallback(callbackquery.Prefix("sel"), Select), callbackQueryGroup)
	d.AddHandlerToGroup(handlers.NewCallback(callbackquery.Prefix("all"), All), callbackQueryGroup)
	d.AddHandlerToGroup(handlers.NewCallback(callbackquery.Prefix("ignore"), Ignore), callbackQueryGroup)
	d.AddHandlerToGroup(handlers.NewCallback(callbackquery.Prefix("config"), ConfigPanel), callbackQueryGroup)
	d.AddHandlerToGroup(handlers.NewCallback(callbackquery.Equal("stats"), Stats), callbackQueryGroup)
	d.AddHandlerToGroup(handlers.NewCallback(callbackquery.Prefix("admenu"), AdminMenuCallback), callbackQueryGroup)
	d.AddHandlerToGroup(handlers.NewCallback(callbackquery.Prefix("index"), CbIndex), callbackQueryGroup)
	d.AddHandlerToGroup(handlers.NewCallback(callbackquery.Prefix("flt"), FilterSearch), callbackQueryGroup)
	d.AddHandlerToGroup(handlers.NewCallback(callbackquery.Prefix("sug"), Autofilter), callbackQueryGroup)
	d.AddHandlerToGroup(handlers.NewCallback(callbackquery.Prefix("cards"), MovieCardCallback), callbackQueryGroup)
	d.AddHandlerToGroup(handlers.NewCallback(callbackquery.Prefix("card"), MovieCardCallback), callbackQueryGroup)
	d.AddHandlerToGroup(handlers.NewCallback(callbackquery.Prefix("browse"), BrowseCallback), callbackQueryGroup)

	d.AddHandlerToGroup(handlers.NewMessage(exthandlers.ChatIds(env.Int64s("FILE_CHANNELS")), NewFile), miscHandlerGroup)
	d.AddHandlerToGroup(handlers.NewChatJoinRequest(func(cjr *gotgbot.ChatJoinRequest) bool { return true }, HandleJoinRequest), joinRequestGroup)

	d.AddHandlerToGroup(handlers.NewMessage(message.All, conversation.MessageHandler), middleWareGroup)
	d.AddHandlerToGroup(exthandlers.NewAllUpdates(EditedMovieCaption), miscHandlerGroup)
	d.AddHandlerToGroup(exthandlers.NewAllUpdates(LogUpdate), miscHandlerGroup)

	return d
}

// logFieldsFromContext adds zap fields to logFields about specific update from ctx to help troubleshooting.
func addLogFieldsFromContext(ctx *ext.Context, logFields []zap.Field) []zap.Field {
	switch {
	case ctx.Message != nil:
		logFields = append(logFields,
			zap.Int64("chat_id", ctx.Message.Chat.Id),
			zap.Int64("message_id", ctx.Message.MessageId),
			zap.String("text", ctx.Message.Text),
		)
	case ctx.CallbackQuery != nil:
		logFields = append(logFields,
			zap.String("callback_query_id", ctx.CallbackQuery.Id),
			zap.String("data", ctx.CallbackQuery.Data),
			zap.Int64("chat_id", ctx.CallbackQuery.Message.GetChat().Id),
			zap.Int64("message_id", ctx.CallbackQuery.Message.GetMessageId()),
		)
	case ctx.InlineQuery != nil:
		logFields = append(logFields,
			zap.String("inline_query_id", ctx.InlineQuery.Id),
			zap.String("query", ctx.InlineQuery.Query),
			zap.Int64("from", ctx.InlineQuery.From.Id),
		)
	}

	return logFields
}

// cleanedStack returns stack trace with gotgbot library parts removed to prevent confusion.
// Copied from https://github.com/PaulSonOfLars/gotgbot/blob/v2/ext/dispatcher.go.
func cleanedStack() string {
	return strings.Join(strings.Split(string(debug.Stack()), "\n")[4:], "\n")
}
