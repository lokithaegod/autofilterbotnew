package core

import (
 "fmt"
 "strconv"
 "strings"

 "github.com/Jisin0/autofilterbot/internal/subtitles"
 "github.com/PaulSonOfLars/gotgbot/v2"
 "github.com/PaulSonOfLars/gotgbot/v2/ext"
 "go.uber.org/zap"
)

// NewSubtitle indexes only document posts in the dedicated subtitle channel.
// This handler does not interact with FILE_CHANNELS or the movie file collection.
func NewSubtitle(bot *gotgbot.Bot, ctx *ext.Context) error {
 m:=ctx.EffectiveMessage
 if m==nil || m.Document==nil {return nil}
 title,year,key,ok:=subtitles.Parse(m.Document.FileName)
 if !ok {
  _app.Log.Info("subtitle skipped: filename needs a movie title, year and .srt/.ass/.vtt extension",
   zap.String("filename",m.Document.FileName))
  return nil
 }
 entry:=&subtitles.Entry{
  UniqueID:m.Document.FileUniqueId,
  FileID:m.Document.FileId,
  FileName:m.Document.FileName,
  Title:title,TitleKey:key,Year:year,
  Size:m.Document.FileSize,
  ChatID:m.Chat.Id,MessageID:m.MessageId,
 }
 if err:=_app.DB.SaveSubtitle(entry);err!=nil{
  _app.Log.Warn("subtitle save failed",zap.Error(err))
  return err
 }
 _app.Log.Info("Malayalam subtitle indexed",zap.String("title",title),zap.String("year",year))
 return nil
}

func SubtitlesMenu(bot *gotgbot.Bot,ctx *ext.Context)error{
 if ctx.CallbackQuery!=nil {_,_=ctx.CallbackQuery.Answer(bot,nil)}
 _,err:=bot.SendMessage(ctx.EffectiveChat.Id,
  "📜 Malayalam Subtitles\n\nSearch using:\n/subtitles The Descent 2005\n\nSupported files: .srt, .ass, .vtt\nIf a matching subtitle is saved, tap its result to receive the file.",nil)
 return err
}

func SubtitlesCommand(bot *gotgbot.Bot,ctx *ext.Context)error{
 if ctx.Message==nil {return nil}
 parts:=strings.SplitN(strings.TrimSpace(ctx.Message.Text)," ",2)
 if len(parts)<2 || strings.TrimSpace(parts[1])=="" {return SubtitlesMenu(bot,ctx)}
 if ctx.Message.From==nil {return nil}
 if len(parts[1])>120 {
  _,err:=ctx.Message.Reply(bot,"Please use a shorter movie title.",nil)
  return err
 }
 blocked,err:=_app.DB.IsUserBlocked(ctx.Message.From.Id)
 if err!=nil{return err}
 if blocked{return nil}
 key,year:=subtitles.SearchTerms(parts[1])
 return sendSubtitleResults(bot,ctx.EffectiveChat.Id,ctx.Message.From.Id,key,year)
}

func sendSubtitleResults(bot *gotgbot.Bot,chatID,userID int64,key,year string)error{
 if key=="" {_,err:=bot.SendMessage(chatID,"Please provide a movie title.",nil);return err}
 entries,err:=_app.DB.SearchSubtitles(key,year)
 if err!=nil{return err}
 if len(entries)==0 {
  _,err=bot.SendMessage(chatID,"📜 No Malayalam subtitles found for that movie yet.",nil)
  return err
 }
 rows:=make([][]gotgbot.InlineKeyboardButton,0,len(entries))
 for _,entry:=range entries {
  // FileUniqueId may contain underscores; split the callback at the last underscore.
  data:=fmt.Sprintf("msend|%s_%d",entry.UniqueID,userID)
  if len(data)>64 {continue}
  label:=entry.FileName
  if len([]rune(label))>55 {label=string([]rune(label)[:52])+"..."}
  rows=append(rows,[]gotgbot.InlineKeyboardButton{{Text:"📜 "+label,CallbackData:data}})
 }
 if len(rows)==0{return fmt.Errorf("subtitle callbacks exceed Telegram's 64 byte limit")}
 _,err=bot.SendMessage(chatID,fmt.Sprintf("📜 Malayalam subtitles (%d result(s))",len(rows)),
  &gotgbot.SendMessageOpts{ReplyMarkup:gotgbot.InlineKeyboardMarkup{InlineKeyboard:rows}})
 return err
}

func SubtitleSendCallback(bot *gotgbot.Bot,ctx *ext.Context)error{
 c:=ctx.CallbackQuery
 if c==nil{return nil}
 payload:=strings.TrimPrefix(c.Data,"msend|")
 idx:=strings.LastIndex(payload,"_")
 if idx<=0{_,_=c.Answer(bot,&gotgbot.AnswerCallbackQueryOpts{Text:"Invalid subtitle button",ShowAlert:true});return nil}
 owner,err:=strconv.ParseInt(payload[idx+1:],10,64)
 if err!=nil||owner!=c.From.Id{
  _,_=c.Answer(bot,&gotgbot.AnswerCallbackQueryOpts{Text:"Only the requester can download this subtitle.",ShowAlert:true})
  return nil
 }
 blocked,err:=_app.DB.IsUserBlocked(c.From.Id)
 if err!=nil||blocked{_,_=c.Answer(bot,&gotgbot.AnswerCallbackQueryOpts{Text:"Access unavailable",ShowAlert:true});return nil}
 entry,err:=_app.DB.GetSubtitle(payload[:idx])
 if err!=nil{
  _,_=c.Answer(bot,&gotgbot.AnswerCallbackQueryOpts{Text:"Subtitle no longer available",ShowAlert:true})
  return nil
 }
 _,_=c.Answer(bot,nil)
 _,err=bot.SendDocument(c.Message.GetChat().Id,gotgbot.InputFileByID(entry.FileID),
  &gotgbot.SendDocumentOpts{Caption:"📜 Malayalam subtitle: "+entry.FileName})
 return err
}

func SubtitleMovieCallback(bot *gotgbot.Bot,ctx *ext.Context)error{
 c:=ctx.CallbackQuery
 if c==nil{return nil}
 payload:=strings.TrimPrefix(c.Data,"msmovie|")
 idx:=strings.LastIndex(payload,"_")
 if idx<=0{_,_=c.Answer(bot,&gotgbot.AnswerCallbackQueryOpts{Text:"Invalid movie",ShowAlert:true});return nil}
 index,err:=strconv.Atoi(payload[idx+1:])
 if err!=nil||index<0{return nil}
 result,ok,err:=_app.Cache.Autofilter.Get(payload[:idx])
 if err!=nil||!ok||result.FromUser!=c.From.Id{
  _,_=c.Answer(bot,&gotgbot.AnswerCallbackQueryOpts{Text:"Search expired or unavailable",ShowAlert:true})
  return nil
 }
 blocked,err:=_app.DB.IsUserBlocked(c.From.Id)
 if err!=nil||blocked{return nil}
 cards:=buildMovieCards(result.Files)
 if index>=len(cards){return nil}
 card:=cards[index]
 if card.Year==""{return nil}
 _,_=c.Answer(bot,nil)
 return sendSubtitleResults(bot,c.Message.GetChat().Id,c.From.Id,subtitles.Normalize(card.Title),card.Year)
}
