package core

import (
 "fmt"
 "strings"

 "github.com/Jisin0/autofilterbot/internal/fsub"
 "github.com/PaulSonOfLars/gotgbot/v2"
 "github.com/PaulSonOfLars/gotgbot/v2/ext"
 "go.uber.org/zap"
)

// FsubAdminDashboard is read-only. Only admins see configured channel IDs,
// permission errors or statistics. It never changes channel settings.
func FsubAdminDashboard(bot *gotgbot.Bot, ctx *ext.Context) error {
 if !_app.AuthAdmin(ctx){return nil}
 var userID, chatID int64
 if ctx.Message!=nil{
  if ctx.Message.Chat.Type!="private"{return nil}
  chatID=ctx.Message.Chat.Id
  userID=ctx.Message.From.Id
 }else if ctx.CallbackQuery!=nil{
  chatID=ctx.CallbackQuery.From.Id
  userID=ctx.CallbackQuery.From.Id
  _,_=ctx.CallbackQuery.Answer(bot,nil)
 }else{return nil}
 _=userID
 channels:=_app.Config.GetFsubChannels()
 lines:=[]string{"🔐 ForceSub channel status",""}
 if len(channels)==0{lines=append(lines,"No force-subscribe channels configured.")}
 for i,ch:=range channels{
  label:=ch.Title
  if label==""{label=fmt.Sprintf("Channel %d",i+1)}
  if len([]rune(label))>64{label=string([]rune(label)[:64])}
  line:=fmt.Sprintf("%d. %s (%d)",i+1,label,ch.ID)
  member,err:=bot.GetChatMember(ch.ID,bot.Id,nil)
  if err!=nil {
   line+="\n   ❌ Can't verify bot permissions: "+friendlyFsubError(err)
   _app.Log.Warn("checkfsub: channel membership lookup failed",zap.Int64("channel_id",ch.ID),zap.Error(err))
  }else{
   switch member.(type){
   case gotgbot.ChatMemberAdministrator,gotgbot.ChatMemberOwner:
    line+="\n   ✅ Bot is channel administrator"
   default:
    line+="\n   ⚠️ Bot is not confirmed as channel admin; GetChatMember on other users may fail"
   }
  }
  if ch.InviteLink==""{line+="\n   ⚠️ Missing invite link"}
  if ch.CreatesJoinRequest{line+="\n   ℹ️ Join-request invite; actual membership still required"}
  lines=append(lines,line)
 }
 stats:=fsub.Snapshot()
 lines=append(lines,"","📊 ForceSub statistics (since current bot startup)",
  fmt.Sprintf("Membership-gate attempts: %d | Passed: %d | Blocked: %d | Errors: %d",stats.Checks,stats.Passed,stats.Blocked,stats.Errors),
  fmt.Sprintf("Telegram membership checks: %d | API errors: %d",stats.MembershipChecks,stats.MembershipErrors),
  fmt.Sprintf("Join prompts: %d | Retry actions: %d | Join requests observed: %d",stats.JoinPrompts,stats.Retries,stats.JoinRequests),
  "Counters reset when the bot restarts; no user identities are stored.",
  "Use /checkfsub to recheck channel permissions.")
 text:=strings.Join(lines,"\n")
 if len(text)>3900{text=string([]rune(text)[:3900])+"\n…"}
 _,err:=bot.SendMessage(chatID,text,&gotgbot.SendMessageOpts{
  ReplyMarkup:gotgbot.InlineKeyboardMarkup{InlineKeyboard:[][]gotgbot.InlineKeyboardButton{
   {{Text:"🔄 Recheck ForceSub",CallbackData:"fsubadmin|refresh"}},
  }},
 })
 return err
}

// CheckFsubAdmin is a convenience alias for diagnostic and dashboard output.
func CheckFsubAdmin(bot *gotgbot.Bot,ctx *ext.Context)error{
 return FsubAdminDashboard(bot,ctx)
}
func friendlyFsubError(err error)string{
 if err==nil{return ""}
 s:=strings.ToLower(err.Error())
 switch{
 case strings.Contains(s,"chat not found")||strings.Contains(s,"channel_invalid"):return "Channel is inaccessible to the bot"
 case strings.Contains(s,"not enough rights")||strings.Contains(s,"administrator"):return "Bot lacks required channel permissions"
 case strings.Contains(s,"too many requests")||strings.Contains(s,"retry after"):return "Telegram rate limit; retry later"
 default:return "Telegram membership verification unavailable"
 }
}
