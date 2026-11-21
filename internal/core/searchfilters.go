package core

import (
 "fmt"
 "regexp"
 "sort"
 "strconv"
 "strings"

 "github.com/Jisin0/autofilterbot/internal/autofilter"
 "github.com/Jisin0/autofilterbot/pkg/callbackdata"
 "github.com/PaulSonOfLars/gotgbot/v2"
 "github.com/PaulSonOfLars/gotgbot/v2/ext"
 "go.uber.org/zap"
)

var qualityToken = regexp.MustCompile(`(?i)(^|[^a-z0-9])(480p|720p|1080p|2160p|4k)([^a-z0-9]|$)`)

func qualityMatches(name, quality string) bool {
 if quality == "all" { return true }
 found := qualityToken.FindStringSubmatch(strings.ToLower(name))
 if len(found) < 3 { return false }
 token := found[2]
 if token == "4k" { token = "2160p" }
 return token == quality
}

func languageMatches(name, language string) bool {
 if language == "all" { return true }
 if language == "mal" && strings.Contains(name,"മലയാളം") {return true}
 // Match labels in filenames only; never infer audio track languages.
 aliases := map[string][]string{
  "mal": {"malayalam", "mal", "malyalam"},
  "tam": {"tamil", "tam"},
  "hin": {"hindi", "hin"},
  "eng": {"english", "eng"},
 }
 lower := strings.ToLower(name)
 for _, word := range aliases[language] {
  re := regexp.MustCompile(`(^|[^a-z])` + word + `([^a-z]|$)`)
  if re.MatchString(lower) { return true }
 }
 return false
}

// Subtitle files are independent Telegram documents, not video containers with
// subtitle tracks. Never promise an embedded subtitle based on an MKV filename.
var subtitleExtension = regexp.MustCompile(`(?i)\.(srt|ass|ssa|vtt|sub)package core

import (
 "fmt"
 "regexp"
 "sort"
 "strconv"
 "strings"

 "github.com/Jisin0/autofilterbot/internal/autofilter"
 "github.com/Jisin0/autofilterbot/pkg/callbackdata"
 "github.com/PaulSonOfLars/gotgbot/v2"
 "github.com/PaulSonOfLars/gotgbot/v2/ext"
 "go.uber.org/zap"
)

var qualityToken = regexp.MustCompile(`(?i)(^|[^a-z0-9])(480p|720p|1080p|2160p|4k)([^a-z0-9]|$)`)

func qualityMatches(name, quality string) bool {
 if quality == "all" { return true }
 found := qualityToken.FindStringSubmatch(strings.ToLower(name))
 if len(found) < 3 { return false }
 token := found[2]
 if token == "4k" { token = "2160p" }
 return token == quality
}

func languageMatches(name, language string) bool {
 if language == "all" { return true }
 // Match labels in filenames only; never infer audio track languages.
 aliases := map[string][]string{
  "mal": {"malayalam", "mal", "malyalam"},
  "tam": {"tamil", "tam"},
  "hin": {"hindi", "hin"},
  "eng": {"english", "eng"},
 }
 lower := strings.ToLower(name)
 for _, word := range aliases[language] {
  re := regexp.MustCompile(`(^|[^a-z])` + word + `([^a-z]|$)`)
  if re.MatchString(lower) { return true }
 }
 return false
}

)
var malayalamSubtitleTag = regexp.MustCompile(`(?i)(^|[^a-z])(malayalam|malyalam|mal[._ -]*sub|m[._ -]*sub|msub)([^a-z]|$)`)
func isSubtitleFile(name string) bool { return subtitleExtension.MatchString(strings.TrimSpace(name)) }
func isMalayalamSubtitle(name string) bool {
 if !isSubtitleFile(name) {return false}
 return strings.Contains(name,"മലയാളം")||malayalamSubtitleTag.MatchString(name)
}
func subtitleMatches(name,kind string) bool {
 switch kind {
 case "all": return true
 case "video": return videoExtension.MatchString(strings.TrimSpace(name))
 case "sub": return isSubtitleFile(name)
 case "msub": return isMalayalamSubtitle(name)
 default: return false
 }
}
func matchingSubtitleFiles(pages []autofilter.Files,quality,language,season,kind string) autofilter.Files {
 var out autofilter.Files
 for _,page:=range pages {for _,f:=range page {
  if !qualityMatches(f.FileName,quality)||!languageMatches(f.FileName,language)||!subtitleMatches(f.FileName,kind){continue}
  if season!="all" {
   n,ok:=autofilter.SeasonNumber(f.FileName)
   if !ok||strconv.Itoa(n)!=season {continue}
  }
  out=append(out,f)
 }}
 return out
}

// Season selection runs after quality and language checks; "all" includes movies.
func matchingSeasonFiles(pages []autofilter.Files, quality, language, season string) autofilter.Files {
 return matchingSubtitleFiles(pages,quality,language,season,"all")
}

// Keep older API helper for existing test compatibility.
func matchingFiles(pages []autofilter.Files,quality,language string) autofilter.Files {
 return matchingSeasonFiles(pages,quality,language,"all")
}
func availableSeasons(pages []autofilter.Files) []int {
 seasons:=map[int]bool{}
 for _,page:=range pages {for _,f:=range page {
  if season,ok:=autofilter.SeasonNumber(f.FileName);ok {seasons[season]=true}
 }}
 var out []int
 for n:=range seasons {out=append(out,n)}
 sort.Ints(out)
 return out
}
func filterButton(id string) []gotgbot.InlineKeyboardButton {
 return []gotgbot.InlineKeyboardButton{{Text:"🎬 Filters",CallbackData:filterCallback(id,"all","all","all","all",0)},{Text:"📜 Malayalam Subs",CallbackData:filterCallback(id,"all","all","all","msub",0)}}
}
func filterCallback(id,quality,language,season,kind string,page int)string{
 return fmt.Sprintf("flt|%s_%s_%s_%s_%s_%d",id,quality,language,season,kind,page)
}
func FilterSearch(bot *gotgbot.Bot, ctx *ext.Context) error {
 c := ctx.CallbackQuery
 data := callbackdata.FromString(c.Data)
 if len(data.Args) != 6 && len(data.Args) != 5 { _,_ = c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text:"Invalid filter",ShowAlert:true}); return nil }
 id, q, l, season := data.Args[0], data.Args[1], data.Args[2], data.Args[3]
 kind:="all"
 pageArg:=4
 if len(data.Args)==6 {kind=data.Args[4];pageArg=5}
 validQ := map[string]bool{"all":true,"480p":true,"720p":true,"1080p":true,"2160p":true}
 validL := map[string]bool{"all":true,"mal":true,"tam":true,"hin":true,"eng":true}
 validKind:=map[string]bool{"all":true,"video":true,"sub":true,"msub":true}
 page,err := strconv.Atoi(data.Args[pageArg])
 if err!=nil || !validQ[q] || !validL[l] || !validKind[kind] || page<0 { _,_ = c.Answer(bot,&gotgbot.AnswerCallbackQueryOpts{Text:"Invalid filter",ShowAlert:true}); return nil }
 result,ok,err := _app.Cache.Autofilter.Get(id)
 if err!=nil || !ok { _,_ = c.Answer(bot,&gotgbot.AnswerCallbackQueryOpts{Text:"Search expired. Search again.",ShowAlert:true}); return nil }
 if result.FromUser!=c.From.Id { _,_ = c.Answer(bot,&gotgbot.AnswerCallbackQueryOpts{Text:"Only the search requester can filter results.",ShowAlert:true}); return nil }
 seasons:=availableSeasons(result.Files)
 validSeason:=season=="all"
 for _,n:=range seasons {if strconv.Itoa(n)==season {validSeason=true;break}}
 if !validSeason {_,_=c.Answer(bot,&gotgbot.AnswerCallbackQueryOpts{Text:"Unknown season",ShowAlert:true});return nil}
 matches:=matchingSubtitleFiles(result.Files,q,l,season,kind)
 var keyboard [][]gotgbot.InlineKeyboardButton
 keyboard=append(keyboard,[]gotgbot.InlineKeyboardButton{{Text:"🎞 Quality",CallbackData:"ignore"}})
 var qualityRow []gotgbot.InlineKeyboardButton
 for _, item := range []struct{code,label string}{{"all","All"},{"480p","480p"},{"720p","720p"},{"1080p","1080p"},{"2160p","4K"}} {
  label := item.label
  if q==item.code {label="✅ "+label}
  qualityRow=append(qualityRow,gotgbot.InlineKeyboardButton{Text:label,CallbackData:filterCallback(id,item.code,l,season,kind,0)})
 }
 keyboard=append(keyboard,qualityRow)
 keyboard=append(keyboard,[]gotgbot.InlineKeyboardButton{{Text:"🌐 Language (filename labels)",CallbackData:"ignore"}})
 var languageRow []gotgbot.InlineKeyboardButton
 for _, item := range []struct{code,label string}{{"all","All"},{"mal","Malayalam"},{"tam","Tamil"},{"hin","Hindi"},{"eng","English"}} {
  label := item.label
  if l==item.code {label="✅ "+label}
  languageRow=append(languageRow,gotgbot.InlineKeyboardButton{Text:label,CallbackData:filterCallback(id,q,item.code,season,kind,0)})
 }
 keyboard=append(keyboard,languageRow)
 if len(seasons)>0 {
  keyboard=append(keyboard,[]gotgbot.InlineKeyboardButton{{Text:"📺 Season",CallbackData:"ignore"}})
  choices:=[]gotgbot.InlineKeyboardButton{{Text:"All Seasons",CallbackData:filterCallback(id,q,l,"all",kind,0)}}
  if season=="all" {choices[0].Text="✅ All Seasons"}
  keyboard=append(keyboard,choices)
  // Group the seasons into short rows to keep the Telegram keyboard readable.
  var row []gotgbot.InlineKeyboardButton
  for _,n:=range seasons {
   code:=strconv.Itoa(n)
   label:=fmt.Sprintf("S%02d",n)
   if season==code {label="✅ "+label}
   row=append(row,gotgbot.InlineKeyboardButton{Text:label,CallbackData:filterCallback(id,q,l,code,kind,0)})
   if len(row)==4 {keyboard=append(keyboard,row);row=nil}
  }
  if len(row)>0 {keyboard=append(keyboard,row)}
 }
 keyboard=append(keyboard,[]gotgbot.InlineKeyboardButton{{Text:"📁 File type",CallbackData:"ignore"}})
 // Two buttons per row make even long labels legible on mobile devices.
 choices:=[]struct{code,label string}{
  {"all","All Files"},{"video","🎬 Videos"},{"sub","📄 All Subtitles"},{"msub","📜 Malayalam Subs"},
 }
 for i:=0;i<len(choices);i+=2 {
  var row []gotgbot.InlineKeyboardButton
  for _,choice:=range choices[i:i+2] {
   label:=choice.label
   if kind==choice.code {label="✅ "+label}
   row=append(row,gotgbot.InlineKeyboardButton{Text:label,CallbackData:filterCallback(id,q,l,season,choice.code,0)})
  }
  keyboard=append(keyboard,row)
 }
 const perPage=8
 totalPages := (len(matches)+perPage-1)/perPage
 if totalPages==0 {totalPages=1}
 if page>=totalPages {page=totalPages-1}
 start:=page*perPage
 end:=start+perPage
 if end>len(matches) {end=len(matches)}
 if len(matches)==0 {
  keyboard=append(keyboard,[]gotgbot.InlineKeyboardButton{{Text:"No files match these filters",CallbackData:"ignore"}})
 } else {
  keyboard=append(keyboard,matches[start:end].Process(c.Message.GetChat().Id,bot.Username,_app.Config)...)
 }
 var paging []gotgbot.InlineKeyboardButton
 if page>0 {paging=append(paging,gotgbot.InlineKeyboardButton{Text:"⬅️",CallbackData:filterCallback(id,q,l,season,kind,page-1)})}
 paging=append(paging,gotgbot.InlineKeyboardButton{Text:fmt.Sprintf("%d/%d • %d files",page+1,totalPages,len(matches)),CallbackData:"ignore"})
 if page+1<totalPages {paging=append(paging,gotgbot.InlineKeyboardButton{Text:"➡️",CallbackData:filterCallback(id,q,l,season,kind,page+1)})}
 keyboard=append(keyboard,paging)
 keyboard=append(keyboard,[]gotgbot.InlineKeyboardButton{{Text:"↩️ Original results",CallbackData:fmt.Sprintf("navg|%s_0",id)}})
 _,_,err=c.Message.EditReplyMarkup(bot,&gotgbot.EditMessageReplyMarkupOpts{ReplyMarkup:gotgbot.InlineKeyboardMarkup{InlineKeyboard:keyboard}})
 if err!=nil { _app.Log.Warn("filter: edit markup failed",zap.Error(err)) }
 _,_=c.Answer(bot,nil)
 return nil
}
