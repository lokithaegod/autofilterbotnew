package core

import (
 "fmt"
 "time"

 "github.com/PaulSonOfLars/gotgbot/v2"
 "github.com/PaulSonOfLars/gotgbot/v2/ext"
 "go.uber.org/zap"
)

var publicBotCommands = []gotgbot.BotCommand{
 {Command:"start", Description:"Start the bot"},
 {Command:"help", Description:"How to search and receive files"},
 {Command:"browse", Description:"Browse movies, series and files"},
 {Command:"find", Description:"Advanced file search"},
 {Command:"movie", Description:"Movie information"},
 {Command:"about", Description:"About the bot"},
}

var adminBotCommands = append(append([]gotgbot.BotCommand{},publicBotCommands...),
 gotgbot.BotCommand{Command:"admin",Description:"Open my private admin menu"},
 gotgbot.BotCommand{Command:"dashboard",Description:"Private admin dashboard"},
 gotgbot.BotCommand{Command:"alive",Description:"Check uptime and bot status"},
 gotgbot.BotCommand{Command:"health",Description:"Test MongoDB and Telegram API health"},
 gotgbot.BotCommand{Command:"stats",Description:"Private bot usage statistics"},
 gotgbot.BotCommand{Command:"settings",Description:"Manage bot settings and Force Sub"},
 gotgbot.BotCommand{Command:"index",Description:"Index channel files"},
 gotgbot.BotCommand{Command:"updateindex",Description:"Index new channel posts"},
 gotgbot.BotCommand{Command:"broadcast",Description:"Admin broadcast"},
)

// Telegram only displays admin commands in each configured admin's private chat.
// Server-side checks in every sensitive command remain essential: users can type
// commands manually even when they are absent from their menu.
func registerBotCommands(bot *gotgbot.Bot, adminIDs []int64, log *zap.Logger) {
 if _,err:=bot.SetMyCommands(publicBotCommands,&gotgbot.SetMyCommandsOpts{
  Scope:gotgbot.BotCommandScopeDefault{},
 });err!=nil {log.Warn("command menu: default registration failed",zap.Error(err))}
 if _,err:=bot.SetMyCommands(publicBotCommands,&gotgbot.SetMyCommandsOpts{
  Scope:gotgbot.BotCommandScopeAllPrivateChats{},
 });err!=nil {log.Warn("command menu: private registration failed",zap.Error(err))}
 for _,id:=range adminIDs {
  if id<=0 {continue}
  if _,err:=bot.SetMyCommands(adminBotCommands,&gotgbot.SetMyCommandsOpts{
   Scope:gotgbot.BotCommandScopeChat{ChatId:id},
  });err!=nil {log.Warn("command menu: admin registration failed",zap.Int64("admin",id),zap.Error(err))}
 }
}

func privateAdminOnly(bot *gotgbot.Bot,ctx *ext.Context) bool {
 if !_app.AuthAdmin(ctx) {return false}
 if ctx.EffectiveChat==nil || ctx.EffectiveChat.Type!="private" {
  if ctx.Message!=nil {_,_=ctx.Message.Reply(bot,"🔒 Please open my private chat to use this admin command.",nil)}
  if ctx.CallbackQuery!=nil {_,_=ctx.CallbackQuery.Answer(bot,&gotgbot.AnswerCallbackQueryOpts{Text:"Open the bot privately to view admin information",ShowAlert:true})}
  return false
 }
 return true
}

func AdminMenu(bot *gotgbot.Bot,ctx *ext.Context) error {
 if !privateAdminOnly(bot,ctx) {return nil}
 msg:="🔐 Admin menu\n\nUse the command menu or tap a button below. Only your configured ADMINS accounts can access these controls."
 keyboard:=gotgbot.InlineKeyboardMarkup{InlineKeyboard:[][]gotgbot.InlineKeyboardButton{
  {{Text:"📊 Dashboard",CallbackData:"admenu|dashboard"},{Text:"💓 Alive",CallbackData:"admenu|alive"}},
  {{Text:"📈 Stats",CallbackData:"admenu|stats"},{Text:"⚙️ Settings",CallbackData:"admenu|settings"}},
  {{Text:"🩺 Health details",CallbackData:"admenu|health"}},
  {{Text:"📥 Index help",CallbackData:"admenu|index"},{Text:"📣 Broadcast help",CallbackData:"admenu|broadcast"}},
 }}
 _,err:=bot.SendMessage(ctx.EffectiveChat.Id,msg,&gotgbot.SendMessageOpts{ReplyMarkup:keyboard})
 return err
}

func Alive(bot *gotgbot.Bot,ctx *ext.Context) error {
 if !privateAdminOnly(bot,ctx) {return nil}
 msg:=fmt.Sprintf("✅ @%s is alive!\n⏱ Uptime: %s\n🔐 Admin access verified",bot.Username,time.Since(_app.StartTime).Truncate(time.Second))
 _,err:=bot.SendMessage(ctx.EffectiveChat.Id,msg,nil)
 return err
}

func AdminMenuCallback(bot *gotgbot.Bot,ctx *ext.Context) error {
 c:=ctx.CallbackQuery
 if !privateAdminOnly(bot,ctx) {return nil}
 _,_=c.Answer(bot,nil)
 switch c.Data {
 case "admenu|dashboard":
  return adminDashboardReply(bot,c.From.Id)
 case "admenu|alive":
  _,err:=bot.SendMessage(c.From.Id,fmt.Sprintf("✅ @%s is alive!\n⏱ Uptime: %s",bot.Username,time.Since(_app.StartTime).Truncate(time.Second)),nil)
  return err
 case "admenu|stats":
  return sendPrivateStats(bot,c.From.Id)
 case "admenu|health":
  _,err:=bot.SendMessage(c.From.Id,"🩺 Send /health to check Telegram API and MongoDB connectivity.",nil)
  return err
 case "admenu|settings":
  _,err:=bot.SendMessage(c.From.Id,"⚙️ Send /settings here to open the interactive admin panel.",nil)
  return err
 case "admenu|index":
  _,err:=bot.SendMessage(c.From.Id,"📥 Send /index to start a channel batch, or forward the newest channel post to incrementally index files. You can also use /updateindex followed by a post link.",nil)
  return err
 case "admenu|broadcast":
  _,err:=bot.SendMessage(c.From.Id,"📣 Send /broadcast in this private chat to start a broadcast.",nil)
  return err
 }
 return nil
}
