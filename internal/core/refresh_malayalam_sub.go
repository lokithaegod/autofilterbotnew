package core

import (
 "fmt"
 "strings"
 "sync/atomic"

 "github.com/Jisin0/autofilterbot/internal/captiontags"
 "github.com/Jisin0/autofilterbot/internal/functions"
 "github.com/Jisin0/autofilterbot/internal/index"
 "github.com/Jisin0/autofilterbot/pkg/env"
 "github.com/PaulSonOfLars/gotgbot/v2"
 "github.com/PaulSonOfLars/gotgbot/v2/ext"
 "go.uber.org/zap"
)

var malayalamRefreshRunning atomic.Bool

func isAuthorizedCaptionChannel(id int64,channels []int64)bool {
 for _,channel:=range channels{if channel==id{return true}}
 return false
}

// refreshForwardedMalayalamVideo is an alternative to Telegram's unsupported
// bot-account channel-history API. An admin forwards an existing channel video
// into this private chat and replies to that forward with /refreshmalayalamsub.
func refreshForwardedMalayalamVideo(bot *gotgbot.Bot,m *gotgbot.Message,channels []int64)error{
 forwarded:=m.ReplyToMessage
 if forwarded==nil{return nil}
 origin,ok:=forwarded.ForwardOrigin.(gotgbot.MessageOriginChannel)
 if !ok || !isAuthorizedCaptionChannel(origin.Chat.Id,channels){
  _,err:=m.Reply(bot,"Please forward the original captioned video directly from an authorized movie storage channel, then reply to that forwarded video with /refreshmalayalamsub.",nil)
  return err
 }
 var name,uniqueID string
 var size int64
 isVideo:=false
 if forwarded.Video!=nil{
  name=forwarded.Video.FileName
  size=forwarded.Video.FileSize
  uniqueID=forwarded.Video.FileUniqueId
  isVideo=true
 }else if forwarded.Document!=nil{
  name=forwarded.Document.FileName
  size=forwarded.Document.FileSize
  uniqueID=forwarded.Document.FileUniqueId
  isVideo=captiontags.IsVideoUpload(name,false)
 }
 if !isVideo||uniqueID==""||name==""{
  _,err:=m.Reply(bot,"The forwarded post must contain a movie video with a filename, not an SRT or other document.",nil)
  return err
 }
 refreshed, err := _app.DB.SetFileCaptionForMedia(uniqueID, name, size, forwarded.Caption)
 if err != nil {
  return err
 }
 if !captiontags.IsTagged(forwarded.Caption) {
  _, err := m.Reply(bot, fmt.Sprintf("Caption metadata refreshed for %d indexed file(s).", refreshed), nil)
  return err
 }
 linked,err:=_app.DB.SetMalayalamCaptionTagForMedia(uniqueID,name,size,true)
 if err!=nil{
  _app.Log.Warn("manual Malayalam caption refresh failed",zap.Error(err))
  _,sendErr:=m.Reply(bot,"❌ Refresh failed while saving the caption tag. Please check the bot logs.",nil)
  return sendErr
 }
 // One result can be only the forwarded Telegram ID, without a matching
 // indexed record; don't misreport it as a movie-file success.
 if linked<=1{
  _,err=m.Reply(bot,"⚠️ Caption recognized, but no matching indexed movie record was found (exact filename and file size). This video may not be indexed yet.",nil)
  return err
 }
 _,err=m.Reply(bot,fmt.Sprintf("✅ Malayalam Sub caption refreshed!\nVideo: %s\nMatching movie records tagged: %d\n\nSearch for the movie again to see the updated section.",name,linked-1),nil)
 return err
}

// RefreshMalayalamSub never fetches unrestricted channel history, because
// Telegram rejects MessagesGetHistory for bot accounts (BOT_METHOD_INVALID).
// Normal caption posts and edits are already tagged automatically.
func RefreshMalayalamSub(bot *gotgbot.Bot,ctx *ext.Context)error{
 m:=ctx.Message
 if m==nil||m.Chat.Type!="private"{return nil}
 if !_app.AuthAdmin(ctx){return nil}
 channels:=env.Int64s("FILE_CHANNELS")
 if len(channels)==0{
  _,err:=m.Reply(bot,"No movie storage channels configured in FILE_CHANNELS.",nil)
  return err
 }
 if m.ReplyToMessage!=nil{return refreshForwardedMalayalamVideo(bot,m,channels)}
 parts:=strings.Fields(m.Text)
 if len(parts)==1{
  _,err:=m.Reply(bot,"✅ Malayalam Sub caption detection is automatic for new posts and caption edits in your configured movie channel.\n\n⚠️ Telegram does not allow bots to scan the entire channel history, so no historical scan has been performed.\n\nTo refresh an older video:\n1. Forward the captioned video from the movie storage channel to this private chat.\n2. Reply to the forwarded video with /refreshmalayalamsub\n\nYou can also specify one or two channel-post links after the command. Send a fresh movie search to update the displayed buttons.",nil)
  return err
 }
 if len(parts)>3{
  _,err:=m.Reply(bot,"Usage: /refreshmalayalamsub [first-channel-post-link] [last-channel-post-link]\nOr reply to a forwarded captioned video with /refreshmalayalamsub.",nil)
  return err
 }
 first,err:=functions.ParseMessageLink(parts[1])
 if err!=nil{
  _,sendErr:=m.Reply(bot,"Invalid channel-post link. Please forward the captioned movie and reply to it with /refreshmalayalamsub.",nil)
  return sendErr
 }
 chat,err:=first.GetChat(bot)
 if err!=nil{
  _,sendErr:=m.Reply(bot,"Unable to resolve that channel. Forward the video to me instead.",nil)
  return sendErr
 }
 if !isAuthorizedCaptionChannel(chat.Id,channels){
  _,err:=m.Reply(bot,"That channel is not an authorized movie storage channel.",nil)
  return err
 }
 from,to:=first.MessageId,first.MessageId
 if len(parts)==3{
  last,err:=functions.ParseMessageLink(parts[2])
  if err!=nil{
   _,sendErr:=m.Reply(bot,"Invalid last channel-post link.",nil)
   return sendErr
  }
  lastChat,err:=last.GetChat(bot)
  if err!=nil||lastChat.Id!=chat.Id{
   _,sendErr:=m.Reply(bot,"Both links must belong to the same authorized movie storage channel.",nil)
   return sendErr
  }
  to=last.MessageId
 }
 if from<=0||to<from||to-from>=100{
  _,err:=m.Reply(bot,"Please provide an ascending range of at most 100 posts.",nil)
  return err
 }
 if !malayalamRefreshRunning.CompareAndSwap(false,true){
  _,err:=m.Reply(bot,"Another linked-post refresh is already running.",nil)
  return err
 }
 _,err=m.Reply(bot,"🔄 Checking the specified channel post(s); refreshing caption metadata.",nil)
 if err!=nil{malayalamRefreshRunning.Store(false);return err}
 go func(){
  defer malayalamRefreshRunning.Store(false)
  result,scanErr:=index.RefreshChannelCaptions(_app.Ctx,bot.Token,chat.Id,from,to,_app.DB.SetMalayalamCaptionTagForMedia, _app.DB.SetFileCaptionForMedia)
  if scanErr!=nil{
   _app.Log.Warn("linked Malayalam caption refresh failed",zap.Int64("chat_id",chat.Id),zap.Error(scanErr))
   _,_=bot.SendMessage(m.Chat.Id,"❌ Could not read those channel posts with the bot's Telegram permissions. Forward the captioned video to this chat and reply /refreshmalayalamsub instead.",nil)
   return
  }
  resultText:=fmt.Sprintf("✅ Post check finished.\nPosts checked: %d\nPosts marked Malayalam Sub: %d\nIndexed file IDs tagged: %d.\nIndexed captions refreshed: %d.",result.Scanned,result.Captioned,result.Linked,result.CaptionsRefreshed)
  if result.Captioned==0{resultText+="\nNo Malayalam Sub labels found in the specified messages."}
  _,_=bot.SendMessage(m.Chat.Id,resultText,nil)
 }()
 return nil
}
