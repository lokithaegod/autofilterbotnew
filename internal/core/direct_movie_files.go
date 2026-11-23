package core

import (
 "fmt"
 "regexp"
 "sort"
 "strings"

 "github.com/Jisin0/autofilterbot/internal/autofilter"
 "github.com/PaulSonOfLars/gotgbot/v2"
 "github.com/PaulSonOfLars/gotgbot/v2/ext"
 "go.uber.org/zap"
)

const directPageSize=7

var englishSubMarker=regexp.MustCompile(`(?i)(?:^|[^a-z0-9])(?:english[ ._-]*(?:subs?|subtitles?)|eng[ ._-]*(?:subs?|subtitles?)|(?:subs?|subtitles?)[ ._-]*english)(?:[^a-z0-9]|$)`)
var englishSRTMarker=regexp.MustCompile(`(?i)(?:^|[^a-z0-9])(?:english|eng)(?:[^a-z0-9]|$)`)
// Tamil dub labels are release claims. A Tamil subtitle or audio-language hint
// alone is not evidence of a Tamil-dubbed video.
var tamilDubMarker=regexp.MustCompile(`(?i)(?:^|[^a-z0-9])(?:tamil|tam)[ ._-]*(?:dub|dubbed)(?:[^a-z0-9]|$)`)

func isTamilDubbedVideo(name string)bool{
 if isMovieSubtitleDocument(name){return false}
 return tamilDubMarker.MatchString(name)||strings.Contains(name,"தமிழ் டப்")
}

var qualityMarker=regexp.MustCompile(`(?i)(?:^|[^0-9])(2160|1080|720|480|360)p(?:[^a-z0-9]|$)`)

type directMovieGroups struct {
 MalayalamVideos autofilter.Files
 MalayalamSubs autofilter.Files
 EnglishVideos autofilter.Files
 EnglishSubs autofilter.Files
 TamilVideos autofilter.Files
}

func isMovieSubtitleDocument(name string)bool{
 return isSubtitleFile(name)||strings.HasSuffix(strings.ToLower(strings.TrimSpace(name)),".zip")
}

func isEnglishMarked(name string)bool{
 if englishSubMarker.MatchString(name){return true}
 if isMovieSubtitleDocument(name) && englishSRTMarker.MatchString(name) {return true}
 return false
}

func fileQuality(name string)int{
 m:=qualityMarker.FindStringSubmatch(name)
 if len(m)<2{return 9999}
 switch m[1]{
 case "360":return 360
 case "480":return 480
 case "720":return 720
 case "1080":return 1080
 case "2160":return 2160
 }
 return 9999
}

func sortDirectVideos(files autofilter.Files){
 sort.SliceStable(files,func(i,j int)bool{
  a,b:=fileQuality(files[i].FileName),fileQuality(files[j].FileName)
  if a!=b{return a<b}
  return strings.ToLower(files[i].FileName)<strings.ToLower(files[j].FileName)
 })
}

func sortDirectSubs(files autofilter.Files){
 sort.SliceStable(files,func(i,j int)bool{
  return strings.ToLower(files[i].FileName)<strings.ToLower(files[j].FileName)
 })
}

// The owner confirms regular non-Tamil movie releases include English subtitles.
// Malayalam-approved files take precedence. Explicitly Tamil-dubbed video
// releases stay in their own section regardless of subtitle availability.
// Preserve all remaining search results in the English section instead of
// dropping files when removing the former Other/Unverified tab.
func groupDirectMovieFiles(pages []autofilter.Files,tagged map[string]bool)directMovieGroups {
 var g directMovieGroups
 for _,page:=range pages{for _,f:=range page{
  subtitle:=isMovieSubtitleDocument(f.FileName)
  switch{
  case isManuallySuitableMalayalamFile(f.FileName,tagged[f.UniqueId]):
   if subtitle{g.MalayalamSubs=append(g.MalayalamSubs,f)}else{g.MalayalamVideos=append(g.MalayalamVideos,f)}
  case (videoExtension.MatchString(f.FileName)||f.FileType=="video") && isTamilDubbedVideo(f.FileName):
   g.TamilVideos=append(g.TamilVideos,f)
  default:
   if subtitle{g.EnglishSubs=append(g.EnglishSubs,f)}else{g.EnglishVideos=append(g.EnglishVideos,f)}
  }
 }}
 sortDirectVideos(g.MalayalamVideos)
 sortDirectSubs(g.MalayalamSubs)
 sortDirectVideos(g.EnglishVideos)
 sortDirectSubs(g.EnglishSubs)
 sortDirectVideos(g.TamilVideos)
 return g
}

func (g directMovieGroups)malayalam()autofilter.Files{
 files:=make(autofilter.Files,0,len(g.MalayalamVideos)+len(g.MalayalamSubs))
 files=append(files,g.MalayalamVideos...)
 return append(files,g.MalayalamSubs...)
}
func (g directMovieGroups)english()autofilter.Files{
 files:=make(autofilter.Files,0,len(g.EnglishVideos)+len(g.EnglishSubs))
 files=append(files,g.EnglishVideos...)
 return append(files,g.EnglishSubs...)
}
func (g directMovieGroups)selected(kind string)autofilter.Files{
 switch kind{
 case "ml":return g.malayalam()
 case "en":return g.english()
 case "ta":return g.TamilVideos
 case "other":return g.english() // Backward-compatible with previously sent buttons.
 }
 return nil
}
func (g directMovieGroups)firstKind()string{
 if len(g.MalayalamVideos)+len(g.MalayalamSubs)>0{return "ml"}
 if len(g.EnglishVideos)+len(g.EnglishSubs)>0{return "en"}
 if len(g.TamilVideos)>0{return "ta"}
 return "en"
}

func directFileCallback(id,kind string,page int)string{
 return fmt.Sprintf("direct|%s_%s_%d",id,kind,page)
}

func directFileKeyboard(id string,g directMovieGroups,kind string,page int,chatID int64,username string,opts autofilter.ProcessFilesOptions)[][]gotgbot.InlineKeyboardButton{
 if kind!="ml"&&kind!="en"&&kind!="ta"&&kind!="other"{kind=g.firstKind()}
 if kind=="other"{kind="en"} // Older callback compatibility
 selected:=g.selected(kind)
 if len(selected)==0{kind=g.firstKind();selected=g.selected(kind)}
 maxPages:=(len(selected)+directPageSize-1)/directPageSize
 if maxPages<1{maxPages=1}
 if page<0{page=0}
 if page>=maxPages{page=maxPages-1}
 result:=make([][]gotgbot.InlineKeyboardButton,0,14)
 label:="✅ Malayalam Sub & Files 👇"
 if kind=="en"{label="🇬🇧 English Sub Files"}
 if kind=="ta"{label="🎙️ Tamil Dubbed Files"}
 if kind=="other"{kind="en";label="🇬🇧 English Sub Files"}
 result=append(result,[]gotgbot.InlineKeyboardButton{{Text:fmt.Sprintf("%s (%d)",label,len(selected)),CallbackData:"ignore"}})
 start:=page*directPageSize
 end:=start+directPageSize
 if end>len(selected){end=len(selected)}
 if start<end{
  // Use the same file-delivery links and ForceSub gate as the existing bot.
  result=append(result,selected[start:end].Process(chatID,username,opts)...)
 }else{
  result=append(result,[]gotgbot.InlineKeyboardButton{{Text:"No files in this section",CallbackData:"ignore"}})
 }
 if maxPages>1{
  nav:=make([]gotgbot.InlineKeyboardButton,0,3)
  if page>0{nav=append(nav,gotgbot.InlineKeyboardButton{Text:"⬅️",CallbackData:directFileCallback(id,kind,page-1)})}
  nav=append(nav,gotgbot.InlineKeyboardButton{Text:fmt.Sprintf("Page %d/%d",page+1,maxPages),CallbackData:"ignore"})
  if page+1<maxPages{nav=append(nav,gotgbot.InlineKeyboardButton{Text:"➡️",CallbackData:directFileCallback(id,kind,page+1)})}
  result=append(result,nav)
 }
 if n:=len(g.MalayalamVideos)+len(g.MalayalamSubs);n>0&&kind!="ml"{
  result=append(result,[]gotgbot.InlineKeyboardButton{{Text:fmt.Sprintf("✅ Malayalam Sub & Files 👇 (%d)",n),CallbackData:directFileCallback(id,"ml",0)}})
 }
 if n:=len(g.EnglishVideos)+len(g.EnglishSubs);n>0&&kind!="en"{
  result=append(result,[]gotgbot.InlineKeyboardButton{{Text:fmt.Sprintf("🇬🇧 English Sub Files (%d)",n),CallbackData:directFileCallback(id,"en",0)}})
 }
 if len(g.TamilVideos)>0&&kind!="ta"{
  result=append(result,[]gotgbot.InlineKeyboardButton{{Text:fmt.Sprintf("🎙️ Tamil Dubbed Files (%d)",len(g.TamilVideos)),CallbackData:directFileCallback(id,"ta",0)}})
 }
 result=append(result,[]gotgbot.InlineKeyboardButton{{Text:"📁 Original file filters",CallbackData:fmt.Sprintf("navg|%s_0",id)}})
 return result
}

func directMovieGroupsFromDB(pages []autofilter.Files)(directMovieGroups,error){
 ids:=make([]string,0)
 for _,p:=range pages{for _,f:=range p{ids=append(ids,f.UniqueId)}}
 tags,err:=_app.DB.FindMalayalamCaptionTags(ids)
 if err!=nil{return directMovieGroups{},err}
 return groupDirectMovieFiles(pages,tags),nil
}

// DirectFileCallback navigates a single result's inline language sections.
// Only the original search requester may change the view.
func DirectFileCallback(bot *gotgbot.Bot,ctx *ext.Context)error{
 c:=ctx.CallbackQuery
 if c==nil{return nil}
 parts:=strings.Split(c.Data,"|")
 if len(parts)!=2{return nil}
 args:=strings.Split(parts[1],"_")
 if len(args)!=3{return nil}
 id,kind:=args[0],args[1]
 var page int
 if _,err:=fmt.Sscanf(args[2],"%d",&page);err!=nil||page<0{return nil}
 session,ok,err:=_app.Cache.Autofilter.Get(id)
 if err!=nil||!ok{_,_=c.Answer(bot,&gotgbot.AnswerCallbackQueryOpts{Text:"Search expired. Search again.",ShowAlert:true});return nil}
 if session.FromUser!=c.From.Id{_,_=c.Answer(bot,&gotgbot.AnswerCallbackQueryOpts{Text:"Only the requester can use these buttons.",ShowAlert:true});return nil}
 blocked,err:=_app.DB.IsUserBlocked(c.From.Id)
 if err!=nil||blocked{_,_=c.Answer(bot,&gotgbot.AnswerCallbackQueryOpts{Text:"Search temporarily unavailable.",ShowAlert:true});return nil}
 if kind!="ml"&&kind!="en"&&kind!="ta"&&kind!="other"{_,_=c.Answer(bot,nil);return nil}
 if kind=="other"{kind="en"} // Old button messages keep working
 if hasEpisodes(session.Files){_,_=c.Answer(bot,nil);return nil}
 groups,err:=directMovieGroupsFromDB(session.Files)
 if err!=nil{
  _app.Log.Warn("direct files: tag lookup unavailable",zap.Error(err))
  _,_=c.Answer(bot,&gotgbot.AnswerCallbackQueryOpts{Text:"Subtitle labels unavailable. Please retry.",ShowAlert:true})
  return nil
 }
 keyboard:=directFileKeyboard(id,groups,kind,page,c.Message.GetChat().Id,bot.Username,_app.Config)
 _,_,err=c.Message.EditReplyMarkup(bot,&gotgbot.EditMessageReplyMarkupOpts{ReplyMarkup:gotgbot.InlineKeyboardMarkup{InlineKeyboard:keyboard}})
 if err!=nil{_app.Log.Warn("direct files: edit markup failed",zap.Error(err))}
 _,_=c.Answer(bot,nil)
 return nil
}
