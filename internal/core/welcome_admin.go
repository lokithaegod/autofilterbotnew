package core

import (
 "fmt"
 "html"
 "net/url"
 "strings"

 "github.com/Jisin0/autofilterbot/internal/button"
 "github.com/PaulSonOfLars/gotgbot/v2"
 "github.com/PaulSonOfLars/gotgbot/v2/ext"
)

// These admin-only commands update the existing MongoDB-backed welcome config.
// HTML is escaped so plain text cannot accidentally break the Telegram HTML renderer.
func SetWelcome(bot *gotgbot.Bot, ctx *ext.Context) error {
 if !_app.AuthAdmin(ctx) { return nil }
 parts:=strings.SplitN(ctx.Message.Text," ",2)
 if len(parts)<2 || strings.TrimSpace(parts[1])=="" {
  _,err:=ctx.Message.Reply(bot,"Usage: /setwelcome Your welcome message (plain text, up to 3000 characters).",nil)
  return err
 }
 message:=strings.TrimSpace(parts[1])
 if len([]rune(message))>3000 {
  _,err:=ctx.Message.Reply(bot,"Welcome text is too long (max 3000 characters).",nil)
  return err
 }
 if err:=_app.DB.UpdateConfig(bot.Id,"start_text",html.EscapeString(message));err!=nil {
  _,e:=ctx.Message.Reply(bot,"Could not save welcome message. Check MongoDB connection.",nil)
  return e
 }
 _app.RefreshConfig()
 _,err:=ctx.Message.Reply(bot,"✅ Welcome message saved. Send /start to preview it.",nil)
 return err
}

func ResetWelcome(bot *gotgbot.Bot, ctx *ext.Context) error {
 if !_app.AuthAdmin(ctx) {return nil}
 if err:=_app.DB.ResetConfig(bot.Id,"start_text");err!=nil {
  _,e:=ctx.Message.Reply(bot,"Could not reset welcome text.",nil);return e
 }
 _app.RefreshConfig()
 _,err:=ctx.Message.Reply(bot,"✅ Default welcome text restored.",nil)
 return err
}

func SetWelcomeButton(bot *gotgbot.Bot, ctx *ext.Context) error {
 if !_app.AuthAdmin(ctx) {return nil}
 parts:=strings.SplitN(ctx.Message.Text," ",2)
 if len(parts)<2 {
  _,err:=ctx.Message.Reply(bot,"Usage: /setwelcomebutton Button label | https://example.com",nil);return err
 }
 fields:=strings.SplitN(strings.TrimSpace(parts[1]),"|",2)
 if len(fields)!=2 {
  _,err:=ctx.Message.Reply(bot,"Usage: /setwelcomebutton Button label | https://example.com",nil);return err
 }
 label,link:=strings.TrimSpace(fields[0]),strings.TrimSpace(fields[1])
 parsed,parseErr:=url.Parse(link)
 if label==""||len([]rune(label))>55||len(link)>500||parseErr!=nil||parsed.Scheme!="https"||parsed.Host==""||parsed.User!=nil {
  _,err:=ctx.Message.Reply(bot,"Use a label of 1–55 characters and a valid https:// link.",nil);return err
 }
 var rows [][]button.InlineKeyboardButton
 if len(_app.Config.StartButtons)==0 {
  for _,row:=range _app.Config.GetStartMessage(bot.Username).Keyboard {
   var converted []button.InlineKeyboardButton
   for _,entry:=range row {converted=append(converted,entry)}
   rows=append(rows,converted)
  }
 } else {
  rows=append(rows,_app.Config.StartButtons...)
 }
 if len(rows)>=10 {
  _,err:=ctx.Message.Reply(bot,"Maximum of 10 welcome button rows reached. Use /resetwelcomebuttons to restore defaults.",nil);return err
 }
 rows=append(rows,[]button.InlineKeyboardButton{{Text:label,Url:link}})
 if err:=_app.DB.UpdateConfig(bot.Id,"start_buttons",rows);err!=nil {
  _,e:=ctx.Message.Reply(bot,"Could not save welcome button.",nil);return e
 }
 _app.RefreshConfig()
 _,err:=ctx.Message.Reply(bot,fmt.Sprintf("✅ Added welcome button: %s",label),nil)
 return err
}

func ResetWelcomeButtons(bot *gotgbot.Bot, ctx *ext.Context) error {
 if !_app.AuthAdmin(ctx) {return nil}
 if err:=_app.DB.ResetConfig(bot.Id,"start_buttons");err!=nil {
  _,e:=ctx.Message.Reply(bot,"Could not reset welcome buttons.",nil);return e
 }
 _app.RefreshConfig()
 _,err:=ctx.Message.Reply(bot,"✅ Default welcome buttons restored.",nil)
 return err
}
