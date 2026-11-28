package core

import (
 "fmt"
 "os"
 "net/url"
 "github.com/Jisin0/autofilterbot/internal/captiontags"
 "regexp"
 "strconv"
 "strings"
 "unicode/utf8"

 "github.com/Jisin0/autofilterbot/internal/autofilter"
 "github.com/Jisin0/autofilterbot/pkg/callbackdata"
 "github.com/PaulSonOfLars/gotgbot/v2"
 "github.com/PaulSonOfLars/gotgbot/v2/ext"
 "go.uber.org/zap"
)

// Movie cards group release variants only when a title and year can be read
// reliably. Episode files remain separate; an unparseable name stays a single
// card rather than being incorrectly attached to another film.
var movieYear = regexp.MustCompile(`(?i)^(.+?)[ ._\-(\[]+((?:19|20)[0-9]{2})(?:[^0-9]|$)`)
var movieWord = regexp.MustCompile(`[^\pL\pM\pN]+`)
// Strip known channel branding before the actual movie title. This avoids
// duplicate cards for e.g. "Inception.2010" and "@CC Links Inception.2010".
var movieLeadingCC = regexp.MustCompile(`(?i)^(?:[._ -]*\[cc\][._ -]*|[._ -]*@cc[._ -]*links[._ -]*|[._ -]*@cc_links[._ -]*|[._ -]*@moviezday[._ -]*)+`)

// Malayalam subtitle archives are downloadable documents, like Malayalam SRTs.
// Only a clear Malayalam name on a .zip is sufficient; generic ZIPs remain "More Files".
func isMalayalamSubtitleArchive(name string) bool {
 lower:=strings.ToLower(strings.TrimSpace(name))
 if !strings.HasSuffix(lower,".zip"){return false}
 return strings.Contains(lower,"മലയാളം") || strings.Contains(lower,"malayalam")
}

// Malayalam section includes explicit owner tags, and Malayalam subtitle
// files recognized by their filename. Forwarded SRT tags do not require
// "Malayalam" to be written in the file name.
func isManuallySuitableMalayalamFile(filename string, tagged bool) bool {
 if tagged{return true}
 return isMalayalamSubtitle(filename)||isMalayalamSubtitleArchive(filename)
}

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
 match:=movieYear.FindStringSubmatch(movieLeadingCC.ReplaceAllString(name,""))
 if len(match)!=3 {return "","","",false}
 // Drop initial file group labels such as [Group] for prettier titles.
 raw:=strings.TrimSpace(strings.Trim(movieLeadingCC.ReplaceAllString(match[1],""),"._- []()"))
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
   name:=sourceFileName(file.File)
   title,year,key,ok:=titleFromFilename(name)
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
   case isSubtitleFile(name) || isMalayalamSubtitleArchive(name):
    c.Subs=append(c.Subs,file)
    if isMalayalamSubtitle(name) || isMalayalamSubtitleArchive(name){c.MalayalamSubs++}
   case captiontags.IsVideoUpload(name,file.FileType=="video"):
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

func showMovieCards(cards []movieCard) bool {
	count:=0
	for _,c:= range cards {
		if c.Year=="" {continue}
		for _,file:=range c.files() {
			if canHaveMovieAudio(file.File) {count++;break}
		}
		if count>=2 {return true}
	}
	return false
}

func (c movieCard) files() autofilter.Files {
	files:=make(autofilter.Files,0,len(c.Videos)+len(c.Subs)+len(c.Other))
	files=append(files,c.Videos...)
	files=append(files,c.Subs...)
	return append(files,c.Other...)
}

func cardLabel(c movieCard) string {
	title:=clippedLabel(c.Title,45)
	if c.Year!="" {title+=" ("+c.Year+")"}
	return "🎬 "+title
}

func cardAudio(files autofilter.Files) []string {
	var labels []string
	for _,code:=range captiontags.AudioCodes {
		for _,file:=range files {
			if file.Audio[code] {
				labels=append(labels,captiontags.LanguageLabel(code))
				break
			}
		}
	}
	return labels
}

func cardsCallback(id string,page int)string{return fmt.Sprintf("cards|%s_%d",id,page)}
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
  label:=fmt.Sprintf("%s · %d files",cardLabel(card),count)
  if audio:=cardAudio(card.files());len(audio)>0 {
   audioLabel:=strings.Join(audio," + ")
   if len(audio)>2 {audioLabel=fmt.Sprintf("%s + %d",audio[0],len(audio)-1)}
   label=fmt.Sprintf("%s · %s",cardLabel(card),audioLabel)
  }
  keyboard=append(keyboard,[]gotgbot.InlineKeyboardButton{{Text:clippedLabel(label,75),CallbackData:cardModeCallback(id,i,0,"film")}})

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
 // Keep caption-based manual claims separate from audio-verified matches.
 // Only files within this movie card are inspected, and the original file
 // collection and file-delivery buttons remain unchanged.
 movieFiles:=make(autofilter.Files,0,len(card.Videos)+len(card.Other)+len(card.Subs))
 movieFiles=append(movieFiles,card.Videos...)
 movieFiles=append(movieFiles,card.Other...)
 movieFiles=append(movieFiles,card.Subs...)
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
  // A Malayalam SRT is a downloadable subtitle document, not a video claim.
  // Videos are suitable only when the owner added the explicit caption tag.
  if isSubtitleFile(f.FileName) || isMalayalamSubtitleArchive(f.FileName) {
   // Admin-forwarded SRTs may not say Malayalam in the original filename.
   // An explicit owner tag takes precedence over filename heuristics.
   if isManuallySuitableMalayalamFile(f.FileName,tagged[f.UniqueId]) {captionTagged=append(captionTagged,f)} else {captionOther=append(captionOther,f)}
  } else if isManuallySuitableMalayalamFile(f.FileName,tagged[f.UniqueId]) {
   captionTagged=append(captionTagged,f)
  } else {
   captionOther=append(captionOther,f)
  }
 }
 all:=make(autofilter.Files,0,len(card.Videos)+len(card.Subs)+len(card.Other))
 switch kind{
 case "tagged":
  all=append(all,captionTagged...)
 case "other":
  all=append(all,captionOther...)
 default:
  all=append(all,card.Videos...)
  all=append(all,card.Subs...)
  all=append(all,card.Other...)
 }
 // Show only the two requested choices when a movie card is opened.
 // Tagged videos are manually confirmed by captions, not by SRT timing analysis.
 if kind=="all" {
  keyboard=append(keyboard,[]gotgbot.InlineKeyboardButton{
   {Text:fmt.Sprintf("✅ Malayalam Sub Files (%d)",len(captionTagged)),CallbackData:cardModeCallback(id,index,0,"tagged")},
  })
  keyboard=append(keyboard,[]gotgbot.InlineKeyboardButton{
   {Text:fmt.Sprintf("🎬 More Files (%d)",len(captionOther)),CallbackData:cardModeCallback(id,index,0,"other")},
  })
 } else {
  selected:="✅ Malayalam Sub Files"
  if kind=="other"{selected="🎬 More Files"}
  keyboard=append(keyboard,[]gotgbot.InlineKeyboardButton{{Text:selected,CallbackData:"ignore"}})
  const perPage=7
  maxPages:=(len(all)+perPage-1)/perPage
  if maxPages==0{maxPages=1}
  if page>=maxPages{page=maxPages-1}
  if page<0{page=0}
  start:=page*perPage
  end:=start+perPage
  if end>len(all){end=len(all)}
  keyboard=append(keyboard,all[start:end].ProcessMovie(chatID,username,opts)...)
  var nav []gotgbot.InlineKeyboardButton
  if page>0{nav=append(nav,gotgbot.InlineKeyboardButton{Text:"⬅️",CallbackData:cardModeCallback(id,index,page-1,kind)})}
  nav=append(nav,gotgbot.InlineKeyboardButton{Text:fmt.Sprintf("%d/%d • %d files",page+1,maxPages,len(all)),CallbackData:"ignore"})
  if page+1<maxPages{nav=append(nav,gotgbot.InlineKeyboardButton{Text:"➡️",CallbackData:cardModeCallback(id,index,page+1,kind)})}
  keyboard=append(keyboard,nav)
  keyboard=append(keyboard,[]gotgbot.InlineKeyboardButton{{Text:"↩️ File categories",CallbackData:cardModeCallback(id,index,0,"all")}})
 }
 keyboard=append(keyboard,[]gotgbot.InlineKeyboardButton{{Text:"↩️ Back to movie cards",CallbackData:cardsCallback(id,index/6)}})
 keyboard=append(keyboard,[]gotgbot.InlineKeyboardButton{{Text:"📁 Original file results",CallbackData:fmt.Sprintf("navg|%s_0",id)}})
 return keyboard
}

func filmKeyboard(id string,index,page int,card movieCard,groups directMovieGroups,kind string,chatID int64,username string,opts autofilter.ProcessFilesOptions) [][]gotgbot.InlineKeyboardButton {
	rows:=[][]gotgbot.InlineKeyboardButton{{{Text:cardLabel(card),CallbackData:"ignore"}}}
	if poster:=posterButton(id,index,card);len(poster)>0 { rows=append(rows,poster) }
	rows=append(rows,movieSectionKeyboard(groups,kind,page,chatID,username,opts,card.Year!="",func(kind string,page int)string{
		return cardModeCallback(id,index,page,kind)
	})...)
	return append(rows,[]gotgbot.InlineKeyboardButton{{Text:"↩️ Back to movie cards",CallbackData:cardsCallback(id,index/6)}})
}

func posterButton(id string,index int,card movieCard) []gotgbot.InlineKeyboardButton {
	if card.Year=="" || os.Getenv("OMDB_API_KEY")=="" {return nil}
	return []gotgbot.InlineKeyboardButton{{Text:"View poster",CallbackData:cardModeCallback(id,index,0,"poster")}}
}

func sendMoviePoster(bot *gotgbot.Bot,c *gotgbot.CallbackQuery,card movieCard) error {
	key:=os.Getenv("OMDB_API_KEY")
	if key=="" || card.Year=="" {
		_,err:=c.Answer(bot,&gotgbot.AnswerCallbackQueryOpts{Text:"Poster unavailable",ShowAlert:true})
		return err
	}
	info,err:=lookupMovieInfo(_app.Ctx,key,card.Title,card.Year)
	if err!=nil || info.Response!="True" || info.Year!=card.Year || !validPosterURL(info.Poster) {
		_,err=c.Answer(bot,&gotgbot.AnswerCallbackQueryOpts{Text:"Poster unavailable",ShowAlert:true})
		return err
	}
	_,_=c.Answer(bot,nil)
	_,err=bot.SendPhoto(c.Message.GetChat().Id,gotgbot.InputFileByURL(info.Poster),&gotgbot.SendPhotoOpts{Caption:cardLabel(card)})
	if err!=nil {
		_,err=bot.SendMessage(c.Message.GetChat().Id,cardLabel(card)+"\n"+info.Poster,nil)
	}
	return err
}

func validPosterURL(raw string) bool {
	poster,err:=url.Parse(raw)
	return err==nil && poster.Scheme=="https" && poster.Hostname()!="" && poster.User==nil
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
  switch kind {
  case "poster":
   return sendMoviePoster(bot,c,cards[index])
  case "film","ml","ta","en":
   card:=cards[index]
   groups,err:=directMovieGroupsFromDB([]autofilter.Files{card.files()})
   if err!=nil {
    _app.Log.Warn("movie card sections unavailable",zap.Error(err))
    keyboard=movieDetailKeyboard(result.UniqueId,index,page,card,c.Message.GetChat().Id,bot.Username,"all",_app.Config)
   } else {
    if kind=="film" && parseMovieSearch(result.Query).audio=="tam" && len(groups.TamilVideos)>0 {kind="ta"}
    keyboard=filmKeyboard(result.UniqueId,index,page,card,groups,kind,c.Message.GetChat().Id,bot.Username,_app.Config)
   }
  case "all","tagged","other":
   keyboard=movieDetailKeyboard(result.UniqueId,index,page,cards[index],c.Message.GetChat().Id,bot.Username,kind,_app.Config)
  default:
   _,_=c.Answer(bot,&gotgbot.AnswerCallbackQueryOpts{Text:"Invalid file type",ShowAlert:true})
   return nil
  }
 }else{
  if page>=(len(cards)+5)/6 {_,_=c.Answer(bot,&gotgbot.AnswerCallbackQueryOpts{Text:"Page not found",ShowAlert:true});return nil}
  keyboard=movieCardsKeyboard(result.UniqueId,cards,page)
 }
 _,_,err=c.Message.EditReplyMarkup(bot,&gotgbot.EditMessageReplyMarkupOpts{ReplyMarkup:gotgbot.InlineKeyboardMarkup{InlineKeyboard:keyboard}})
 if err!=nil {_app.Log.Warn("movie cards: edit markup failed",zap.Error(err))}
 _,_=c.Answer(bot,nil)
 return nil
}

