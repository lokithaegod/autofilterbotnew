package core

import (
 "fmt"
 "path/filepath"
 "strings"

 "github.com/Jisin0/autofilterbot/internal/captiontags"
 "github.com/Jisin0/autofilterbot/pkg/env"
 "github.com/PaulSonOfLars/gotgbot/v2"
 "github.com/PaulSonOfLars/gotgbot/v2/ext"
 "go.uber.org/zap"
)

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
  _,err:=m.Reply(bot,"⚠️ Forward the original video or .srt directly from your authorized movie storage channel. Unverified forwarding origins cannot mark database files.",nil)
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
  _,err:=m.Reply(bot,"Only original movie videos and .srt subtitle documents can be marked. ZIPs and unrelated files are ignored.",nil)
  return err
 }
 matched,newlyTagged,err:=_app.DB.TagIndexedMalayalamMedia(uniqueID,filename,size)
 if err!=nil{
  _app.Log.Warn("forwarded Malayalam file tag failed",zap.Error(err),zap.String("file_name",filename))
  _,sendErr:=m.Reply(bot,"❌ Could not verify or save the Malayalam Sub tag. Please try again. No duplicate movie file was created.",nil)
  return sendErr
 }
 if matched==0{
  _,err=m.Reply(bot,fmt.Sprintf("⚠️ Not indexed yet.\nFile: %s\n\nI couldn't find the original file in the database. Please index it from your authorized movie channel first. Nothing was added or tagged.",filename),nil)
  return err
 }
 label:="movie video"
 if kind=="srt"{label="Malayalam subtitle"}
 if newlyTagged==0{
  _,err=m.Reply(bot,fmt.Sprintf("✅ Already added to Malayalam Sub Files.\nFile: %s\nExisting indexed records: %d\n\nNo duplicate was added.",filename,matched),nil)
  return err
 }
 _app.Log.Info("admin forwarded Malayalam file tagged",zap.String("kind",kind),zap.Int("matched_indexed_records",matched),zap.Int("newly_tagged",newlyTagged))
 _,err=m.Reply(bot,fmt.Sprintf("✅ %s added to Malayalam Sub Files.\nFile: %s\nNewly tagged records: %d\nExisting indexed records: %d\n\nSearch for the movie again to see the updated section. No duplicate file was added.",label,filename,newlyTagged,matched),nil)
 return err
}
