package core

import (
 "fmt"
 "path/filepath"
 "regexp"
 "strings"
 "sync"
 "time"

 "github.com/Jisin0/autofilterbot/internal/captiontags"
 "github.com/Jisin0/autofilterbot/pkg/env"
 "github.com/PaulSonOfLars/gotgbot/v2"
 "github.com/PaulSonOfLars/gotgbot/v2/ext"
 "go.uber.org/zap"
)

// Telegram limits how quickly a bot can message one chat. Forwarding an album
// generates many updates together, so serialize the confirmations to avoid a
// burst of sendMessage requests. File tagging remains independent of replies.
var forwardedMalayalamReplyMu sync.Mutex
var forwardedMalayalamLastReply time.Time
var telegramTokenInError = regexp.MustCompile(`bot[0-9]+:[A-Za-z0-9_-]+`)

func replyToForwardedMalayalam(bot *gotgbot.Bot, m *gotgbot.Message, text string) error {
 forwardedMalayalamReplyMu.Lock()
 defer forwardedMalayalamReplyMu.Unlock()
 if remaining:=1100*time.Millisecond-time.Since(forwardedMalayalamLastReply);remaining>0 {
  time.Sleep(remaining)
 }
 // Set before the request, even if it fails, to avoid retry storms.
 forwardedMalayalamLastReply=time.Now()
 _,err:=m.Reply(bot,text,nil)
 if err!=nil {
  // Bot API errors may embed the full token in the request URL.
  clean:=telegramTokenInError.ReplaceAllString(err.Error(),"bot[REDACTED]")
  _app.Log.Warn("forwarded Malayalam confirmation send failed",zap.String("error",clean),zap.Int64("chat_id",m.Chat.Id))
 }
 return err
}

// ForwardedMalayalamKind recognizes only video files and SRT subtitle documents.
// Forwarding is an explicit manual assertion by the admin, not audio verification.
func ForwardedMalayalamKind(name string,video bool) string{
 if captiontags.IsVideoUpload(name,video){return "video"}
 if strings.EqualFold(filepath.Ext(name),".srt"){return "srt"}
 return ""
}

// MarkForwardedMalayalam accepts forwarded media only from configured movie
// storage channels and only in an admin's private chat. It never saves a
// second file or changes an existing movie record.
func MarkForwardedMalayalam(bot *gotgbot.Bot,ctx *ext.Context)error{
 m:=ctx.Message
 if m==nil||m.From==nil||m.Chat.Type!="private"||m.ForwardOrigin==nil{return nil}
 // Ordinary users must never be able to label a movie release.
 if !_app.AuthAdmin(ctx){return nil}
 origin,ok:=m.ForwardOrigin.(gotgbot.MessageOriginChannel)
 if !ok||!isAuthorizedCaptionChannel(origin.Chat.Id,env.Int64s("FILE_CHANNELS")){
  _,err:=replyToForwardedMalayalam(bot,m,"⚠️ Forward the original video or .srt directly from your authorized movie storage channel. Unverified forwarding origins cannot mark database files.")
  return err
 }
 var filename,uniqueID string
 var size int64
 video:=false
 if m.Video!=nil{
  filename=m.Video.FileName
  uniqueID=m.Video.FileUniqueId
  size=m.Video.FileSize
  video=true
 }else if m.Document!=nil{
  filename=m.Document.FileName
  uniqueID=m.Document.FileUniqueId
  size=m.Document.FileSize
 }
 kind:=ForwardedMalayalamKind(filename,video)
 if kind==""||uniqueID==""||filename==""||size<=0{
  _,err:=replyToForwardedMalayalam(bot,m,"Only original movie videos and .srt subtitle documents can be marked. ZIPs and unrelated files are ignored.")
  return err
 }
 matched,newlyTagged,err:=_app.DB.TagIndexedMalayalamMedia(uniqueID,filename,size)
 if err!=nil{
  _app.Log.Warn("forwarded Malayalam file tag failed",zap.Error(err),zap.String("file_name",filename))
  _,sendErr:=replyToForwardedMalayalam(bot,m,"❌ Could not verify or save the Malayalam Sub tag. Please try again. No duplicate movie file was created.")
  return sendErr
 }
 if matched==0{
  _,err=replyToForwardedMalayalam(bot,m,fmt.Sprintf("⚠️ Not indexed yet.\nFile: %s\n\nI couldn't find the original file in the database. Please index it from your authorized movie channel first. Nothing was added or tagged.",filename))
  return err
 }
 label:="movie video"
 if kind=="srt"{label="Malayalam subtitle"}
 if newlyTagged==0{
  _,err=replyToForwardedMalayalam(bot,m,fmt.Sprintf("✅ Already added to Malayalam Sub Files.\nFile: %s\nExisting indexed records: %d\n\nNo duplicate was added.",filename,matched))
  return err
 }
 _app.Log.Info("admin forwarded Malayalam file tagged",zap.String("kind",kind),zap.Int("matched_indexed_records",matched),zap.Int("newly_tagged",newlyTagged))
 _,err=replyToForwardedMalayalam(bot,m,fmt.Sprintf("✅ %s added to Malayalam Sub Files.\nFile: %s\nNewly tagged records: %d\nExisting indexed records: %d\n\nSearch for the movie again to see the updated section. No duplicate file was added.",label,filename,newlyTagged,matched))
 return err
}
