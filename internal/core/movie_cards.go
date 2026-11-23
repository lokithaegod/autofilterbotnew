package core

import (
 "fmt"
 "regexp"
 "strconv"
 "strings"
 "unicode/utf8"

 "github.com/Jisin0/autofilterbot/internal/autofilter"
 "github.com/Jisin0/autofilterbot/internal/subtitles"
 "github.com/Jisin0/autofilterbot/pkg/callbackdata"
 "github.com/PaulSonOfLars/gotgbot/v2"
 "github.com/PaulSonOfLars/gotgbot/v2/ext"
 "go.uber.org/zap"
)

// Movie cards group release variants only when a title and year can be read
// reliably. Episode files remain separate; an unparseable name stays a single
// card rather than being incorrectly attached to another film.
var movieYear = regexp.MustCompile(`(?i)^(.+?)[ ._\-(\[]+((?:19|20)[0-9]{2})(?:[^0-9]|$)`)
var movieWord = regexp.MustCompile(`[^a-z0-9]+`)

type movieCard struct {
 Title string
 Year string
 Videos autofilter.Files
 Subs autofilter.Files
 Other autofilter.Files
 MalayalamSubs int
}

func titleFromFilename(name string)(title,year,key string,ok bool){
 if _,isEpisode:=autofilter.SeasonNumber(name);isEpisode {return "","","",false}
 match:=movieYear.FindStringSubmatch(name)
 if len(match)!=3 {return "","","",false}
 // Drop initial file group labels such as [Group] for prettier titles.
 raw:=strings.TrimSpace(strings.Trim(match[1],"._- []()"))
 words:=movieWord.ReplaceAllString(strings.ToLower(raw)," ")
 words=strings.Join(strings.Fields(words)," ")
 if words=="" {return "","","",false}
 display:=strings.ReplaceAll(strings.ReplaceAll(raw,"."," "),"_"," ")
 return strings.Join(strings.Fields(display)," "),match[2],words+"|"+match[2],true
}

func buildMovieCards(pages []autofilter.Files) []movieCard {
 var cards []movieCard
 lookup:=make(map[string]int)
 unique:=0
 for _,page:=range pages {
  for _,file:=range page {
   title,year,key,ok:=titleFromFilename(file.FileName)
   if !ok {
    // Without a title + year match, don't risk merging unrelated releases.
    key=fmt.Sprintf("single|%d",unique)
    unique++
    title=file.FileName
   }
   i,exists:=lookup[key]
   if !exists {
    i=len(cards);lookup[key]=i
    cards=append(cards,movieCard{Title:title,Year:year})
   }
   c:=&cards[i]
   switch {
   case isSubtitleFile(file.FileName):
    c.Subs=append(c.Subs,file)
    if isMalayalamSubtitle(file.FileName){c.MalayalamSubs++}
   case videoExtension.MatchString(file.FileName):
    c.Videos=append(c.Videos,file)
   default:
    c.Other=append(c.Other,file)
   }
  }
 }
 return cards
}

func hasEpisodes(pages []autofilter.Files)bool {
 for _,page:=range pages {for _,f:=range page {
  if _,ok:=autofilter.SeasonNumber(f.FileName);ok{return true}
 }}
 return false
}

func clippedLabel(s string,max int) string {
 if utf8.RuneCountInString(s)<=max{return s}
 r:=[]rune(s)
 return string(r[:max-1])+"…"
}

func cardLabel(c movieCard) string {
 title:=c.Title
 if c.Year!="" {title+=" ("+c.Year+")"}
 label:="🎬 "+clippedLabel(title,37)
 if c.MalayalamSubs>0 {label+=" ✅📜"}
 return label
}

func cardsCallback(id string,page int)string{return fmt.Sprintf("cards|%s_%d",id,page)}
func cardCallback(id string,index,page int)string{return fmt.Sprintf("card|%s_%d_%d",id,index,page)}
func cardModeCallback(id string,index,page int,kind string)string{return fmt.Sprintf("card|%s_%d_%d_%s",id,index,page,kind)}

func movieCardsKeyboard(id string,cards []movieCard,page int) [][]gotgbot.InlineKeyboardButton {
 const perPage=6
 maxPages:=(len(cards)+perPage-1)/perPage
 if maxPages<1 {maxPages=1}
 if page<0{page=0}
 if page>=maxPages{page=maxPages-1}
 start:=page*perPage
 end:=start+perPage
 if end>len(cards){end=len(cards)}
 keyboard:=[][]gotgbot.InlineKeyboardButton{
  {{Text:"📁 Show all files & filters",CallbackData:fmt.Sprintf("navg|%s_0",id)}},
 }
 for i:=start;i<end;i++ {
  card:=cards[i]
  count:=len(card.Videos)+len(card.Subs)+len(card.Other)
  label:=fmt.Sprintf("%s • %d files",cardLabel(card),count)
  keyboard=append(keyboard,[]gotgbot.InlineKeyboardButton{{Text:clippedLabel(label,75),CallbackData:cardCallback(id,i,0)}})
  if movieSubtitleAvailable(card) {
   keyboard=append(keyboard,[]gotgbot.InlineKeyboardButton{{Text:"📜 Malayalam Subtitle",CallbackData:fmt.Sprintf("msmovie|%s_%d",id,i)}})
  }
 }
 var nav []gotgbot.InlineKeyboardButton
 if page>0{nav=append(nav,gotgbot.InlineKeyboardButton{Text:"⬅️",CallbackData:cardsCallback(id,page-1)})}
 nav=append(nav,gotgbot.InlineKeyboardButton{Text:fmt.Sprintf("🎬 %d/%d",page+1,maxPages),CallbackData:"ignore"})
 if page+1<maxPages {nav=append(nav,gotgbot.InlineKeyboardButton{Text:"➡️",CallbackData:cardsCallback(id,page+1)})}
 keyboard=append(keyboard,nav)
 return keyboard
}

func movieDetailKeyboard(id string,index,page int,card movieCard,chatID int64,username,kind string,opts autofilter.ProcessFilesOptions) [][]gotgbot.InlineKeyboardButton {
 var keyboard [][]gotgbot.InlineKeyboardButton
 keyboard=append(keyboard,[]gotgbot.InlineKeyboardButton{{Text:cardLabel(card),CallbackData:"ignore"}})
 if card.MalayalamSubs>0 {
  keyboard=append(keyboard,[]gotgbot.InlineKeyboardButton{{Text:"✅ Malayalam subtitle file available",CallbackData:"ignore"}})
 }
 if movieSubtitleAvailable(card) {
  keyboard=append(keyboard,[]gotgbot.InlineKeyboardButton{{Text:"📜 Download Malayalam Subtitle",CallbackData:fmt.Sprintf("msmovie|%s_%d",id,index)}})
 }
 // Keep caption-based manual claims separate from audio-verified matches.
 // Only files within this movie card are inspected, and the original file
 // collection and file-delivery buttons remain unchanged.
 movieFiles:=make(autofilter.Files,0,len(card.Videos)+len(card.Other))
 movieFiles=append(movieFiles,card.Videos...)
 movieFiles=append(movieFiles,card.Other...)
 ids:=make([]string,0,len(movieFiles))
 for _,f:=range movieFiles {ids=append(ids,f.UniqueId)}
 tagged:=make(map[string]bool)
 var tagErr error
 if _app!=nil && _app.DB!=nil {
  tagged,tagErr=_app.DB.FindMalayalamCaptionTags(ids)
 }
 if tagErr!=nil{
  if _app!=nil && _app.Log!=nil{_app.Log.Warn("movie cards: caption tags lookup failed",zap.Error(tagErr))}
  if kind=="tagged"||kind=="other"{
   return [][]gotgbot.InlineKeyboardButton{{{Text:"⚠️ Subtitle tags temporarily unavailable",CallbackData:"ignore"}},
    {{Text:"↩️ Back to movie cards",CallbackData:cardsCallback(id,index/6)}}}
  }
 }
 captionTagged:=make(autofilter.Files,0)
 captionOther:=make(autofilter.Files,0)
 for _,f:=range movieFiles{
  if tagged[f.UniqueId] {captionTagged=append(captionTagged,f)} else {captionOther=append(captionOther,f)}
 }
 all:=make(autofilter.Files,0,len(card.Videos)+len(card.Subs)+len(card.Other))
 switch kind{
 case "msub":
  for _,f:=range card.Subs {if isMalayalamSubtitle(f.FileName){all=append(all,f)}}
 case "tagged":
  all=append(all,captionTagged...)
 case "other":
  all=append(all,captionOther...)
 default:
  all=append(all,card.Videos...)
  all=append(all,card.Subs...)
  all=append(all,card.Other...)
 }
 var modeRow []gotgbot.InlineKeyboardButton
 allLabel:="🎬 All Files"
 if kind=="all"{allLabel="✅ "+allLabel}
 modeRow=append(modeRow,gotgbot.InlineKeyboardButton{Text:allLabel,CallbackData:cardModeCallback(id,index,0,"all")})
 if card.MalayalamSubs>0 {
  subLabel:="📜 Malayalam Subs"
  if kind=="msub"{subLabel="✅ "+subLabel}
  modeRow=append(modeRow,gotgbot.InlineKeyboardButton{Text:subLabel,CallbackData:cardModeCallback(id,index,0,"msub")})
 }
 keyboard=append(keyboard,modeRow)
 const perPage=7
 maxPages:=(len(all)+perPage-1)/perPage
 if maxPages==0{maxPages=1}
 if page>=maxPages{page=maxPages-1}
 if page<0{page=0}
 start:=page*perPage
 end:=start+perPage
 if end>len(all){end=len(all)}
 // Each file uses the pre-existing /start deep link, including Force Sub.
 keyboard=append(keyboard,all[start:end].Process(chatID,username,opts)...)
 var nav []gotgbot.InlineKeyboardButton
 if page>0{nav=append(nav,gotgbot.InlineKeyboardButton{Text:"⬅️",CallbackData:cardModeCallback(id,index,page-1,kind)})}
 nav=append(nav,gotgbot.InlineKeyboardButton{Text:fmt.Sprintf("%d/%d • %d options",page+1,maxPages,len(all)),CallbackData:"ignore"})
 if page+1<maxPages{nav=append(nav,gotgbot.InlineKeyboardButton{Text:"➡️",CallbackData:cardModeCallback(id,index,page+1,kind)})}
 keyboard=append(keyboard,nav)
 // These labels indicate user-provided captions, not verified audio timing.
 tagLabel:=fmt.Sprintf("✅ Malayalam Sub Available (%d)",len(captionTagged))
 otherLabel:=fmt.Sprintf("🎬 Other Files (%d)",len(captionOther))
 if kind=="tagged"{tagLabel="☑️ "+tagLabel}
 if kind=="other"{otherLabel="☑️ "+otherLabel}
 if tagErr==nil {
  keyboard=append(keyboard,[]gotgbot.InlineKeyboardButton{
   {Text:tagLabel,CallbackData:cardModeCallback(id,index,0,"tagged")},
  })
  keyboard=append(keyboard,[]gotgbot.InlineKeyboardButton{
   {Text:otherLabel,CallbackData:cardModeCallback(id,index,0,"other")},
  })
 }
 keyboard=append(keyboard,[]gotgbot.InlineKeyboardButton{{Text:"↩️ Back to movie cards",CallbackData:cardsCallback(id,index/6)}})
 keyboard=append(keyboard,[]gotgbot.InlineKeyboardButton{{Text:"📁 Original file results",CallbackData:fmt.Sprintf("navg|%s_0",id)}})
 return keyboard
}

func MovieCardCallback(bot *gotgbot.Bot,ctx *ext.Context)error{
 c:=ctx.CallbackQuery
 data:=callbackdata.FromString(c.Data)
 expected:=2
 if data.Path[0]=="card"{expected=3}
 if data.Path[0]=="card"&&len(data.Args)==4{expected=4}
 if len(data.Args)!=expected {_,_=c.Answer(bot,&gotgbot.AnswerCallbackQueryOpts{Text:"Invalid card",ShowAlert:true});return nil}
 result,ok,err:=_app.Cache.Autofilter.Get(data.Args[0])
 if err!=nil||!ok {_,_=c.Answer(bot,&gotgbot.AnswerCallbackQueryOpts{Text:"Search expired. Search again.",ShowAlert:true});return nil}
 if result.FromUser!=c.From.Id {_,_=c.Answer(bot,&gotgbot.AnswerCallbackQueryOpts{Text:"Only the requester can open movie cards.",ShowAlert:true});return nil}
 blocked,err:=_app.DB.IsUserBlocked(c.From.Id)
 if err!=nil||blocked{_,_=c.Answer(bot,&gotgbot.AnswerCallbackQueryOpts{Text:"Access unavailable",ShowAlert:true});return nil}
 cards:=buildMovieCards(result.Files)
 pageIndex:=expected-1
 if data.Path[0]=="card"{pageIndex=2}
 page,err:=strconv.Atoi(data.Args[pageIndex])
 if err!=nil||page<0 {_,_=c.Answer(bot,&gotgbot.AnswerCallbackQueryOpts{Text:"Invalid page",ShowAlert:true});return nil}
 var keyboard [][]gotgbot.InlineKeyboardButton
 if data.Path[0]=="card"{
  index,err:=strconv.Atoi(data.Args[1])
  if err!=nil||index<0||index>=len(cards){_,_=c.Answer(bot,&gotgbot.AnswerCallbackQueryOpts{Text:"Movie not found",ShowAlert:true});return nil}
  kind:="all"
  if expected==4 {kind=data.Args[3]}
  if kind!="all"&&kind!="msub"&&kind!="tagged"&&kind!="other"{_,_=c.Answer(bot,&gotgbot.AnswerCallbackQueryOpts{Text:"Invalid file type",ShowAlert:true});return nil}
  keyboard=movieDetailKeyboard(result.UniqueId,index,page,cards[index],c.Message.GetChat().Id,bot.Username,kind,_app.Config)
 }else{
  if page>=(len(cards)+5)/6 {_,_=c.Answer(bot,&gotgbot.AnswerCallbackQueryOpts{Text:"Page not found",ShowAlert:true});return nil}
  keyboard=movieCardsKeyboard(result.UniqueId,cards,page)
 }
 _,_,err=c.Message.EditReplyMarkup(bot,&gotgbot.EditMessageReplyMarkupOpts{ReplyMarkup:gotgbot.InlineKeyboardMarkup{InlineKeyboard:keyboard}})
 if err!=nil {_app.Log.Warn("movie cards: edit markup failed",zap.Error(err))}
 _,_=c.Answer(bot,nil)
 return nil
}

// Subtitle availability is queried from the independent collection; movie records are not changed.
func movieSubtitleAvailable(card movieCard)bool{
 if card.Year=="" || _app==nil || _app.DB==nil{return false}
 available,err:=_app.DB.SubtitleExists(subtitles.Normalize(card.Title),card.Year)
 return err==nil&&available
}
