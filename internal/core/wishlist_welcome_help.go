package core

import (
 "fmt"
 "math/rand"
 "strings"

 "github.com/Jisin0/autofilterbot/internal/button"
 "github.com/PaulSonOfLars/gotgbot/v2"
 "github.com/PaulSonOfLars/gotgbot/v2/ext"
)

const downloadHelpText="📥 HOW TO DOWNLOAD\n\n1. Search for a movie by title.\n2. Open ✅ Malayalam Sub & Files 👇, 🇬🇧 English Sub Files, or 🎙️ Tamil Dubbed Files.\n3. Tap your desired file quality (480p, 720p, or 1080p).\n4. If asked, join the required ForceSub channel and tap the retry link.\n5. Tap Start in private chat to receive the file.\n\nFor Malayalam subtitle files, download the .srt file and select it in your video player.\n\n⚠️ File messages can auto-delete; save only content you're authorized to keep."

func downloadHelpButton()[]gotgbot.InlineKeyboardButton{
 return []gotgbot.InlineKeyboardButton{{Text:"📥 How to Download",CallbackData:"downloadhelp"}}
}
func HowToDownload(bot *gotgbot.Bot,ctx *ext.Context)error{
 if ctx.CallbackQuery!=nil{
  c:=ctx.CallbackQuery
  _,_=c.Answer(bot,nil)
  _,err:=bot.SendMessage(c.From.Id,downloadHelpText,nil)
  if err!=nil{
   _,_=c.Answer(bot,&gotgbot.AnswerCallbackQueryOpts{Text:"Open the bot's private chat and send /howtodownload.",ShowAlert:true})
  }
  return nil
 }
 _,err:=ctx.Message.Reply(bot,downloadHelpText,nil)
 return err
}

func AddWelcomePhoto(bot *gotgbot.Bot,ctx *ext.Context)error{
 if !privateAdminOnly(bot,ctx){return nil}
 m:=ctx.Message
 var fileID string
 if m.ReplyToMessage!=nil&&len(m.ReplyToMessage.Photo)>0{
  photos:=m.ReplyToMessage.Photo
  fileID=photos[len(photos)-1].FileId
 }
 if fileID=="" {
  args:=ctx.Args();if len(args)>=2{fileID=strings.TrimSpace(args[1])}
 }
 if fileID==""||len(fileID)>500{_,err:=m.Reply(bot,"Reply to a photo with /addwelcomephoto, or use /addwelcomephoto <Telegram photo file ID>.",nil);return err}
 if err:=_app.DB.AddWelcomePhoto(fileID);err!=nil{return err}
 _,err:=m.Reply(bot,"✅ Welcome image added. It will rotate on /start without a file link.",nil);return err
}
func ListWelcomePhotos(bot *gotgbot.Bot,ctx *ext.Context)error{
 if !privateAdminOnly(bot,ctx){return nil}
 photos,err:=_app.DB.ListWelcomePhotos();if err!=nil{return err}
 _,err=ctx.Message.Reply(bot,fmt.Sprintf("🖼️ %d welcome images configured. To remove one, reply with /removewelcomephoto <Telegram photo file ID>.",len(photos)),nil)
 return err
}
func RemoveWelcomePhoto(bot *gotgbot.Bot,ctx *ext.Context)error{
 if !privateAdminOnly(bot,ctx){return nil}
 args:=ctx.Args();if len(args)<2{_,err:=ctx.Message.Reply(bot,"Usage: /removewelcomephoto <Telegram photo file ID>",nil);return err}
 ok,err:=_app.DB.RemoveWelcomePhoto(strings.TrimSpace(args[1]));if err!=nil{return err}
 msg:="Photo not found";if ok{msg="✅ Welcome photo removed"}
 _,err=ctx.Message.Reply(bot,msg,nil);return err
}
func sendRandomWelcome(bot *gotgbot.Bot,ctx *ext.Context)error {
 photos,err:=_app.DB.ListWelcomePhotos()
 if err!=nil||len(photos)==0{return StaticCommands(bot,ctx)}
 photo:=photos[rand.Intn(len(photos))]
 welcome:=_app.Config.GetStartMessage(bot.Username)
 welcome.Format(_app.BasicMessageValues(ctx))
 keyboard:=gotgbot.InlineKeyboardMarkup{InlineKeyboard:button.UnwrapKeyboard(welcome.Keyboard)}
 _,err=bot.SendPhoto(ctx.EffectiveChat.Id,gotgbot.InputFileByID(photo),&gotgbot.SendPhotoOpts{
  Caption:welcome.Text,ParseMode:gotgbot.ParseModeHTML,ReplyMarkup:keyboard,
 })
 if err!=nil{return StaticCommands(bot,ctx)}
 return nil
}
