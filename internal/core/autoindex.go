package core

import (
 "fmt"
 "strings"

 "github.com/Jisin0/autofilterbot/internal/functions"
 "github.com/Jisin0/autofilterbot/internal/index"
 "github.com/Jisin0/autofilterbot/internal/model"
 "github.com/Jisin0/autofilterbot/pkg/conversation"
 "github.com/PaulSonOfLars/gotgbot/v2"
 "github.com/PaulSonOfLars/gotgbot/v2/ext"
 "go.uber.org/zap"
)

const initialIndexScanLimit int64 = 100

// AutoIndexLast indexes new posts through the forwarded final channel post.
// An explicit /updateindex <post link> is also supported.
func AutoIndexLast(bot *gotgbot.Bot, ctx *ext.Context) error {
 if ctx.Message == nil || ctx.Message.From == nil {return nil}
 if !_app.AuthAdmin(ctx) {return nil}
 m := ctx.Message
 var channelID,endID int64
 origin, ok := m.ForwardOrigin.(gotgbot.MessageOriginChannel)
 if !ok && m.ReplyToMessage!=nil {origin,ok=m.ReplyToMessage.ForwardOrigin.(gotgbot.MessageOriginChannel)}
 if ok {channelID=origin.Chat.Id;endID=origin.MessageId}
 if endID==0 {
  split:=strings.Fields(m.Text)
  if len(split)>1 {
   if link,err:=functions.ParseMessageLink(split[1]);err==nil {
    chat,err:=link.GetChat(bot)
    if err!=nil {sendChatErr(bot,m.Chat.Id,err);return nil}
    channelID=chat.Id;endID=link.MessageId
   }
  }
 }
 if endID<=0 || channelID>=0 {
  _,err:=m.Reply(bot,"Forward the last post directly from your channel, or send /updateindex followed by its Telegram post link. The bot must be an admin in that channel.",nil)
  return err
 }
 if m.Chat.Type!="private" {
  _,err:=m.Reply(bot,"Please use /updateindex in my private chat.",nil);return err
 }
 // Never start an overlapping index for the same channel.
 active,err:=_app.DB.GetActiveIndexOperations()
 if err!=nil {_,e:=m.Reply(bot,"Could not verify active index jobs. Try again later.",nil);return e}
 for _,job:=range active {if job.ChannelID==channelID {
   _,e:=m.Reply(bot,"An indexing job is already active for this channel. Finish or pause it first.",nil);return e
 }}
 last,err:=_app.DB.GetIndexCheckpoint(channelID)
 if err!=nil { _app.Log.Warn("autoindex: checkpoint read failed",zap.Error(err));_,e:=m.Reply(bot,"Could not read indexing checkpoint. Try again later.",nil);return e }
 if last>=endID {
  _,e:=m.Reply(bot,fmt.Sprintf("✅ Already indexed through post %d. No new messages.",last),nil);return e
 }
 if last==0 {
  // No known checkpoint from the old indexer. Safely scan a limited recent window.
  last=endID-initialIndexScanLimit
  if last<0 {last=0}
 }
 first:=last+1
 msg,err:=m.Reply(bot,fmt.Sprintf("📥 Indexing channel %d: posts %d–%d (%d positions).",channelID,first,endID,endID-first+1),nil)
 if err!=nil {return err}
 op:=&model.Index{
  ID:functions.RandString(6),
  StartMessageID:first,EndMessageID:endID,CurrentMessageID:first-1,
  ChannelID:channelID,ProgressMessageChatID:msg.Chat.Id,IsPaused:false,
 }
 if err:=_app.DB.NewIndexOperation(op);err!=nil {
  _app.Log.Warn("autoindex: save job failed",zap.Error(err))
  _,e:=m.Reply(bot,"Couldn't create indexing job: database unavailable.",nil);return e
 }
 _app.IndexManager.CancelOperation(op.ID)
 jobCtx,operation:=_app.IndexManager.NewOperation(_app.Ctx,op,_app.DB,_app.Log,bot)
 _app.IndexManager.RunOperation(jobCtx,operation)
 _=index.TDLibChannelIDToPlain(channelID) // channel id is validated for the existing index engine
 return nil
}

// PrivateLastFile detects an admin-forwarded channel post and uses it as an automatic index end.
func PrivateLastFile(bot *gotgbot.Bot, ctx *ext.Context) error {
 m:=ctx.Message
 if m==nil || m.From==nil || conversation.ActiveListeners.HasMatch(m) {return nil}
 if _,ok:=m.ForwardOrigin.(gotgbot.MessageOriginChannel);ok && containsI64(_app.Admins,m.From.Id) {
  return AutoIndexLast(bot,ctx)
 }
 return PrivateAutofilter(bot,ctx)
}
