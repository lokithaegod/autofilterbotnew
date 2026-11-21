package core

import (
 "fmt"
 "regexp"
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

func matchingFiles(pages []autofilter.Files, quality, language string) autofilter.Files {
 var out autofilter.Files
 for _, page := range pages {
  for _, file := range page {
   if qualityMatches(file.FileName, quality) && languageMatches(file.FileName, language) {
    out = append(out, file)
   }
  }
 }
 return out
}

func filterButton(id string) []gotgbot.InlineKeyboardButton {
 return []gotgbot.InlineKeyboardButton{{Text:"🎬 Quality / Language", CallbackData:"flt|"+id+"_all_all_0"}}
}

func filterCallback(id, quality, language string, page int) string {
 return fmt.Sprintf("flt|%s_%s_%s_%d", id, quality, language, page)
}

func FilterSearch(bot *gotgbot.Bot, ctx *ext.Context) error {
 c := ctx.CallbackQuery
 data := callbackdata.FromString(c.Data)
 if len(data.Args) != 4 { _,_ = c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text:"Invalid filter",ShowAlert:true}); return nil }
 id, q, l := data.Args[0], data.Args[1], data.Args[2]
 validQ := map[string]bool{"all":true,"480p":true,"720p":true,"1080p":true,"2160p":true}
 validL := map[string]bool{"all":true,"mal":true,"tam":true,"hin":true,"eng":true}
 page,err := strconv.Atoi(data.Args[3])
 if err!=nil || !validQ[q] || !validL[l] || page<0 { _,_ = c.Answer(bot,&gotgbot.AnswerCallbackQueryOpts{Text:"Invalid filter",ShowAlert:true}); return nil }
 result,ok,err := _app.Cache.Autofilter.Get(id)
 if err!=nil || !ok { _,_ = c.Answer(bot,&gotgbot.AnswerCallbackQueryOpts{Text:"Search expired. Search again.",ShowAlert:true}); return nil }
 if result.FromUser!=c.From.Id { _,_ = c.Answer(bot,&gotgbot.AnswerCallbackQueryOpts{Text:"Only the search requester can filter results.",ShowAlert:true}); return nil }
 matches := matchingFiles(result.Files,q,l)
 var keyboard [][]gotgbot.InlineKeyboardButton
 keyboard=append(keyboard,[]gotgbot.InlineKeyboardButton{{Text:"🎞 Quality",CallbackData:"ignore"}})
 var qualityRow []gotgbot.InlineKeyboardButton
 for _, item := range []struct{code,label string}{{"all","All"},{"480p","480p"},{"720p","720p"},{"1080p","1080p"},{"2160p","4K"}} {
  label := item.label
  if q==item.code {label="✅ "+label}
  qualityRow=append(qualityRow,gotgbot.InlineKeyboardButton{Text:label,CallbackData:filterCallback(id,item.code,l,0)})
 }
 keyboard=append(keyboard,qualityRow)
 keyboard=append(keyboard,[]gotgbot.InlineKeyboardButton{{Text:"🌐 Language (filename labels)",CallbackData:"ignore"}})
 var languageRow []gotgbot.InlineKeyboardButton
 for _, item := range []struct{code,label string}{{"all","All"},{"mal","Malayalam"},{"tam","Tamil"},{"hin","Hindi"},{"eng","English"}} {
  label := item.label
  if l==item.code {label="✅ "+label}
  languageRow=append(languageRow,gotgbot.InlineKeyboardButton{Text:label,CallbackData:filterCallback(id,q,item.code,0)})
 }
 keyboard=append(keyboard,languageRow)
 const perPage=8
 totalPages := (len(matches)+perPage-1)/perPage
 if totalPages==0 {totalPages=1}
 if page>=totalPages {page=totalPages-1}
 start:=page*perPage
 end:=start+perPage
 if end>len(matches) {end=len(matches)}
 if len(matches)==0 {
  keyboard=append(keyboard,[]gotgbot.InlineKeyboardButton{{Text:"No matching filename labels",CallbackData:"ignore"}})
 } else {
  keyboard=append(keyboard,matches[start:end].Process(c.Message.GetChat().Id,bot.Username,_app.Config)...)
 }
 var paging []gotgbot.InlineKeyboardButton
 if page>0 {paging=append(paging,gotgbot.InlineKeyboardButton{Text:"⬅️",CallbackData:filterCallback(id,q,l,page-1)})}
 paging=append(paging,gotgbot.InlineKeyboardButton{Text:fmt.Sprintf("%d/%d • %d files",page+1,totalPages,len(matches)),CallbackData:"ignore"})
 if page+1<totalPages {paging=append(paging,gotgbot.InlineKeyboardButton{Text:"➡️",CallbackData:filterCallback(id,q,l,page+1)})}
 keyboard=append(keyboard,paging)
 keyboard=append(keyboard,[]gotgbot.InlineKeyboardButton{{Text:"↩️ Original results",CallbackData:fmt.Sprintf("navg|%s_0",id)}})
 _,_,err=c.Message.EditReplyMarkup(bot,&gotgbot.EditMessageReplyMarkupOpts{ReplyMarkup:gotgbot.InlineKeyboardMarkup{InlineKeyboard:keyboard}})
 if err!=nil { _app.Log.Warn("filter: edit markup failed",zap.Error(err)) }
 _,_=c.Answer(bot,nil)
 return nil
}
