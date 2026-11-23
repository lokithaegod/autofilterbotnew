package index

import (
 "context"
 "fmt"
 "os"
 "strconv"
 "strings"
 "time"

 "github.com/Jisin0/autofilterbot/internal/captiontags"
 "github.com/amarnathcjd/gogram/telegram"
)

// CaptionScanResult summarizes media posts inspected without touching the
// existing index checkpoints, file documents, or download IDs.
type CaptionScanResult struct {
 Scanned int
 Captioned int
 Linked int
}

// RefreshChannelCaptions reads the latest channel messages through the same
// MTProto bot login already used by the movie indexer. When message IDs are
// supplied, only that inclusive range is scanned. The callback persists tags
// in a separate collection and must be idempotent.
func RefreshChannelCaptions(ctx context.Context, botToken string, channelID int64, startID,endID int64, apply func(string,string,int64,bool)(int,error)) (CaptionScanResult,error) {
 var result CaptionScanResult
 if channelID==0 || botToken=="" {return result,fmt.Errorf("missing channel ID or bot token")}
 if apply==nil{return result,fmt.Errorf("caption tag callback unavailable")}
 appID,appHash:=DefaultAppID,DefaultAppHash
 if val:=os.Getenv("APP_ID");val!=""{
  if parsed,err:=strconv.Atoi(val);err==nil && os.Getenv("APP_HASH")!="" {appID=parsed;appHash=os.Getenv("APP_HASH")}
 }
 client,err:=telegram.NewClient(telegram.ClientConfig{
  AppID:int32(appID),AppHash:appHash,NoUpdates:true,MemorySession:true,
  LogLevel:telegram.LogError,DisableCache:true,
 })
 if err!=nil{return result,err}
 if err=client.LoginBot(botToken);err!=nil{return result,err}
 if _,err=client.GetMe();err!=nil{return result,err}
 channel,err:=getChat(client,channelID)
 if err!=nil{return result,fmt.Errorf("channel access failed: %w",err)}
 deadline,cancel:=context.WithTimeout(ctx,100*time.Second)
 defer cancel()
 // Limit each invocation to 300 fetched posts to avoid flooding Telegram and
 // consuming Render CPU. For old posts, specify the target message IDs.
 const maxPosts=300
 if startID>0 {
  if endID==0{endID=startID}
  if endID<startID || endID-startID+1>maxPosts {
   return result,fmt.Errorf("message range must be ascending and at most %d posts",maxPosts)
  }
  for from:=startID;from<=endID;from+=100 {
   select{case <-deadline.Done():return result,deadline.Err();default:}
   to:=from+99;if to>endID{to=endID}
   ids:=make([]telegram.InputMessage,0,int(to-from+1))
   for n:=from;n<=to;n++{ids=append(ids,&telegram.InputMessageID{ID:int32(n)})}
   history,err:=client.ChannelsGetMessages(channel,ids)
   if err!=nil{return result,err}
   if err=scanCaptionMessages(history,channelID,apply,&result);err!=nil{return result,err}
  }
 }else{
  // No arguments: read the latest 300 posts, in three requests.
  // Historical caption edits on much older posts require an explicit post link.
  offset:=int32(0)
  for page:=0;page<3;page++{
   select{case <-deadline.Done():return result,deadline.Err();default:}
   history,err:=client.MessagesGetHistory(&telegram.MessagesGetHistoryParams{
    Peer:&telegram.InputPeerChannel{ChannelID:channel.ChannelID,AccessHash:channel.AccessHash},
    OffsetID:offset,Limit:100,
   })
   if err!=nil{return result,err}
   msgs:=captionHistoryMessages(history)
   if len(msgs)==0{break}
   if err=scanCaptionMessages(history,channelID,apply,&result);err!=nil{return result,err}
   oldest:=int32(0)
   for _,item:=range msgs{
    if m,ok:=item.(*telegram.MessageObj);ok && (oldest==0||m.ID<oldest){oldest=m.ID}
   }
   if oldest<=0||oldest==offset{break}
   offset=oldest
   if len(msgs)<100{break}
  }
 }
 return result,nil
}

func captionHistoryMessages(history telegram.MessagesMessages) []telegram.Message {
 switch v:=history.(type) {
 case *telegram.MessagesChannelMessages:return v.Messages
 case *telegram.MessagesMessagesObj:return v.Messages
 case *telegram.MessagesMessagesSlice:return v.Messages
 default:return nil
 }
}

func scanCaptionMessages(history telegram.MessagesMessages,channelID int64,apply func(string,string,int64,bool)(int,error),result *CaptionScanResult)error{
 for _,item:=range captionHistoryMessages(history){
  msg,ok:=item.(*telegram.MessageObj)
  if !ok {continue}
  result.Scanned++
  if !captiontags.IsTagged(msg.Message){continue}
  media,ok:=msg.Media.(*telegram.MessageMediaDocument);if !ok{continue}
  doc,ok:=media.Document.(*telegram.DocumentObj);if !ok{continue}
  name:=""
  hasVideoAttribute:=false
  for _,attribute:=range doc.Attributes{
   switch a:=attribute.(type){
   case *telegram.DocumentAttributeFilename:name=a.FileName
   case *telegram.DocumentAttributeVideo:hasVideoAttribute=true
   }
  }
  if !captiontags.IsVideoUpload(name,hasVideoAttribute){continue}
  result.Captioned++
  identity:=fmt.Sprintf("mtproto-caption:%d:%d",channelID,msg.ID)
  linked,err:=apply(identity,strings.TrimSpace(name),int64(doc.Size),true)
  if err!=nil{return err}
  if linked>1 {result.Linked+=linked-1}
 }
 return nil
}
