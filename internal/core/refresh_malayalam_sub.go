package core

import (
 "fmt"
 "strings"
 "sync/atomic"

 "github.com/Jisin0/autofilterbot/internal/functions"
 "github.com/Jisin0/autofilterbot/internal/index"
 "github.com/Jisin0/autofilterbot/pkg/env"
 "github.com/PaulSonOfLars/gotgbot/v2"
 "github.com/PaulSonOfLars/gotgbot/v2/ext"
 "go.uber.org/zap"
)

var malayalamRefreshRunning atomic.Bool

// RefreshMalayalamSub is intentionally admin-only, and only reads channels
// already configured as FILE_CHANNELS. It never reindexes media files.
func RefreshMalayalamSub(bot *gotgbot.Bot,ctx *ext.Context) error {
 m:=ctx.Message
 if m==nil || m.Chat.Type!="private"{return nil}
 if !_app.AuthAdmin(ctx){return nil}
 channels:=env.Int64s("FILE_CHANNELS")
 if len(channels)==0 {
  _,err:=m.Reply(bot,"No movie storage channels configured in FILE_CHANNELS.",nil)
  return err
 }
 parts:=strings.Fields(m.Text)
 if len(parts)>3 {
  _,err:=m.Reply(bot,"Usage:\n/refreshmalayalamsub\n/refreshmalayalamsub <channel-post-link>\n/refreshmalayalamsub <first-link> <last-link>\n\nWithout links, scans up to 300 recent posts per configured movie channel.",nil)
  return err
 }
 var channelID,firstID,lastID int64
 if len(parts)>1{
  first,err:=functions.ParseMessageLink(parts[1])
  if err!=nil{_,e:=m.Reply(bot,"Invalid channel post link. Send a Telegram post link such as https://t.me/c/1234567890/50",nil);return e}
  chat,err:=first.GetChat(bot)
  if err!=nil{return fmt.Errorf("resolve refresh channel: %w",err)}
  channelID=chat.Id
  firstID=first.MessageId
  if len(parts)==3{
   last,err:=functions.ParseMessageLink(parts[2])
   if err!=nil{_,e:=m.Reply(bot,"Invalid last post link.",nil);return e}
   lastChat,err:=last.GetChat(bot)
   if err!=nil{return fmt.Errorf("resolve last post channel: %w",err)}
   if lastChat.Id!=channelID{_,e:=m.Reply(bot,"Both links must belong to the same movie channel.",nil);return e}
   lastID=last.MessageId
  }
  authorized:=false
  for _,id:=range channels{if id==channelID{authorized=true;break}}
  if !authorized{_,e:=m.Reply(bot,"That channel is not one of your authorized movie storage channels.",nil);return e}
  channels=[]int64{channelID}
 }
 if !malayalamRefreshRunning.CompareAndSwap(false,true) {
  _,err:=m.Reply(bot,"A Malayalam Sub refresh is already running. Please use the result of that scan.",nil)
  return err
 }
 status,err:=m.Reply(bot,"🔄 Checking Malayalam Sub captions in your movie storage channel(s). Existing movie records will not be changed.",nil)
 if err!=nil{malayalamRefreshRunning.Store(false);return err}
 go func(){
  defer malayalamRefreshRunning.Store(false)
  total:=index.CaptionScanResult{}
  failed:=make([]string,0)
  for _,id:=range channels{
   result,scanErr:=index.RefreshChannelCaptions(_app.Ctx,bot.Token,id,firstID,lastID,_app.DB.SetMalayalamCaptionTagForMedia)
   total.Scanned+=result.Scanned
   total.Captioned+=result.Captioned
   total.Linked+=result.Linked
   if scanErr!=nil{
    failed=append(failed,fmt.Sprintf("%d: %v",id,scanErr))
    _app.Log.Warn("refresh malayalam captions channel scan failed",zap.Int64("chat_id",id),zap.Error(scanErr))
   }
  }
  resultMessage:=fmt.Sprintf("✅ Malayalam Sub refresh completed\n\nChannel posts checked: %d\nPosts with Malayalam Sub captions: %d\nMatching indexed file IDs refreshed: %d\n\nFresh movie searches will display the updated sections.",total.Scanned,total.Captioned,total.Linked)
  if len(failed)>0{
   resultMessage+="\n\n⚠️ Some channel scans failed:\n"+strings.Join(failed,"\n")
   resultMessage+="\nFor older posts, try /refreshmalayalamsub followed by the post link."
  }
  if total.Captioned==0&&len(failed)==0{
   resultMessage+="\n\nNo marked videos found in the scanned posts. Use a channel-post link for older uploads."
  }
  if _,sendErr:=bot.SendMessage(m.Chat.Id,resultMessage,nil);sendErr!=nil{
   _app.Log.Warn("refresh malayalam captions response failed",zap.Error(sendErr))
  }
  if status!=nil{_,_=status.Delete(bot,nil)}
 }()
 return nil
}
