package core

import (
 "github.com/Jisin0/autofilterbot/internal/captiontags"
 "github.com/Jisin0/autofilterbot/pkg/env"
 "github.com/PaulSonOfLars/gotgbot/v2"
 "github.com/PaulSonOfLars/gotgbot/v2/ext"
 "go.uber.org/zap"
)

// EditedMovieCaption processes edited movie-channel video captions, including
// older posts whose file records were already stored in the movie database.
// Only tags in the independent caption collection are created/deleted.
func EditedMovieCaption(bot *gotgbot.Bot,ctx *ext.Context)error{
 if ctx.Update==nil || ctx.Update.EditedChannelPost==nil{return nil}
 m:=ctx.Update.EditedChannelPost
 authorized:=false
 for _,id:=range env.Int64s("FILE_CHANNELS"){if id==m.Chat.Id{authorized=true;break}}
 if !authorized{
  _app.Log.Debug("edited movie caption ignored: channel not in FILE_CHANNELS",zap.Int64("chat_id",m.Chat.Id))
  return nil
 }
 var uniqueID,name string
 isVideo:=false
 if m.Video!=nil{
  uniqueID=m.Video.FileUniqueId
  name=m.Video.FileName
  isVideo=true
 }else if m.Document!=nil{
  uniqueID=m.Document.FileUniqueId
  name=m.Document.FileName
  isVideo=captiontags.IsVideoUpload(name,false)
 }
 if !isVideo || uniqueID=="" {return nil}
 enabled:=captiontags.IsTagged(m.Caption)
 if err:=_app.DB.SetMalayalamCaptionTag(uniqueID,enabled);err!=nil{
  _app.Log.Warn("edited movie caption tag update failed",zap.Error(err))
  return err
 }
 _app.Log.Info("edited movie caption tag processed",zap.Int64("chat_id",m.Chat.Id),zap.String("file_name",name),zap.Bool("malayalam_sub",enabled))
 return nil
}
