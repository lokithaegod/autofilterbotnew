package core

import (
 "sort"
 "strings"
 "sync"
 "time"
 "unicode"

 "github.com/Jisin0/autofilterbot/internal/functions"
 "github.com/PaulSonOfLars/gotgbot/v2"
)

type suggestedTitle struct {
 Label string
 Query string
 Normal string
}
type scoredTitle struct{suggestedTitle;score int}

var suggestionIndex = struct{
 sync.Mutex
 expire time.Time
 titles []suggestedTitle
}{}
var suggestionsByToken = struct{
 sync.Mutex
 items map[string]suggestionSession
}{items:make(map[string]suggestionSession)}
type suggestionSession struct {
 UserID int64
 ChatID int64
 Titles []suggestedTitle
 Expires time.Time
}

func episodeSuggestionTitle(name string) string {
 // Recognizes "Show.S01E02" and "Show.1x02" without including episode names.
 lower:=strings.ToLower(name)
 loc:=suggestionEpisode.FindStringIndex(lower)
 if loc==nil{return ""}
 name=strings.Trim(name[:loc[0]]," ._-[]()")
 if strings.EqualFold(name,"fri"){return "Fringe"}
 return strings.Join(strings.Fields(strings.NewReplacer("."," ","_"," ","-"," ").Replace(name))," ")
}

func titleCandidate(filename string)(suggestedTitle,bool){
 title,year,_,ok:=titleFromFilename(filename)
 if !ok {
  title=episodeSuggestionTitle(filename)
  if title=="" {return suggestedTitle{},false}
 }
 n:=normalizeTitle(title)
 if len(n)<3||len(n)>85{return suggestedTitle{},false}
 // Filter unusable numeric/codec-only pseudo-titles.
 hasLetter:=false
 for _,c:=range n{if unicode.IsLetter(c){hasLetter=true;break}}
 if !hasLetter{return suggestedTitle{},false}
 label:=title
 query:=title
 if year!="" {label+=" ("+year+")";query+=" "+year}
 return suggestedTitle{Label:label,Query:query,Normal:n},true
}

func uniqueTitlesFromFilenames(files []string)[]suggestedTitle{
 result:=make([]suggestedTitle,0,150)
 seen:=make(map[string]bool)
 for _,filename:=range files{
  c,ok:=titleCandidate(filename)
  if !ok {continue}
  key:=c.Normal+"|"+strings.TrimSpace(strings.TrimPrefix(c.Label,strings.TrimSuffix(c.Label," ("+extractYear(c.Label)+")")))
  // Prefer a year-specific key for movies; deduplicate series.
  if yr:=extractYear(c.Label);yr!=""{key=c.Normal+"|"+yr}else{key=c.Normal}
  if seen[key]{continue}
  seen[key]=true
  result=append(result,c)
 }
 return result
}

func extractYear(label string)string{
 if len(label)>=7&&label[len(label)-1]==')'&&label[len(label)-6]=='('{
  y:=label[len(label)-5:len(label)-1]
  for _,r:=range y{if r<'0'||r>'9'{return ""}}
  return y
 }
 return ""
}

func compactWords(s string)string{return strings.ReplaceAll(s," ","")}

func candidateScore(query,title string)(int,bool){
 if query==""||title==""{return 0,false}
 if query==title{return 0,true}
 q:=compactWords(query);t:=compactWords(title)
 if q==t{return 0,true}
 d:=fuzzyDistance(q,t)
 // Long titles may contain extra words; use the best matching prefix.
 if strings.HasPrefix(t,q) {d=1}
 qw:=strings.Fields(query);tw:=strings.Fields(title)
 if len(qw)>1&&len(tw)>=len(qw){
  // Allow "Once Upon Hollywood" to match "Once Upon a Time in Hollywood".
  // Each query word must occur in the right order. One-letter stopwords
  // do not count as a full match for unrelated filenames.
  pos:=0;matched:=true;penalty:=0
  for _,word:=range qw{
   found:=-1
   for j:=pos;j<len(tw);j++{
    if tw[j]==word||fuzzyDistance(word,tw[j])<=1&&len(word)>=4{found=j;break}
   }
   if found<0{matched=false;break}
   penalty+=found-pos
   pos=found+1
  }
  if matched&&penalty+1<d{d=penalty+1}
 }
 // Short queries require especially tight matches to avoid random results.
 limit:=2
 if len(q)>=10 {limit=len(q)/4+1}
 if len(q)>=20 {limit=len(q)/3+1}
 if d>limit{return 0,false}
 return d,true
}

func rankSuggestions(query string,titles []suggestedTitle)[]suggestedTitle{
 q:=normalizeTitle(query)
 if len(q)<3{return nil}
 var matches []scoredTitle
 for _,c:=range titles{
  score,ok:=candidateScore(q,c.Normal)
  if !ok{continue}
  matches=append(matches,scoredTitle{suggestedTitle:c,score:score})
 }
 sort.SliceStable(matches,func(i,j int)bool{
  if matches[i].score!=matches[j].score{return matches[i].score<matches[j].score}
  if len(matches[i].Normal)!=len(matches[j].Normal){return len(matches[i].Normal)<len(matches[j].Normal)}
  return matches[i].Label<matches[j].Label
 })
 out:=make([]suggestedTitle,0,3)
 used:=map[string]bool{}
 for _,m:=range matches{
  if used[m.Normal]{continue}
  used[m.Normal]=true;out=append(out,m.suggestedTitle)
  if len(out)==3{break}
 }
 return out
}

func SuggestTitles(query string)([]suggestedTitle,error){
 suggestionIndex.Lock()
 if time.Now().Before(suggestionIndex.expire) {
  titles:=suggestionIndex.titles
  suggestionIndex.Unlock()
  return rankSuggestions(query,titles),nil
 }
 suggestionIndex.Unlock()
 filenames,err:=_app.DB.SuggestionFilenames()
 if err!=nil{return nil,err}
 titles:=uniqueTitlesFromFilenames(filenames)
 suggestionIndex.Lock()
 suggestionIndex.titles=titles
 suggestionIndex.expire=time.Now().Add(10*time.Minute)
 suggestionIndex.Unlock()
 return rankSuggestions(query,titles),nil
}

func saveSuggestionSession(userID,chatID int64,titles []suggestedTitle)string {
 token:=functions.RandString(14)
 suggestionsByToken.Lock()
 defer suggestionsByToken.Unlock()
 now:=time.Now()
 for id,s:=range suggestionsByToken.items{if now.After(s.Expires){delete(suggestionsByToken.items,id)}}
 if len(suggestionsByToken.items)>300{
  suggestionsByToken.items=make(map[string]suggestionSession)
 }
 suggestionsByToken.items[token]=suggestionSession{UserID:userID,ChatID:chatID,Titles:append([]suggestedTitle(nil),titles...),Expires:now.Add(12*time.Minute)}
 return token
}

func suggestionFromCallback(raw string,userID,chatID int64)(string,bool){
 // callback format "sug|token_0", never embed arbitrary titles in Telegram's
 // 64-byte callback or allow a user to use another user's result.
 parts:=strings.Split(raw,"|")
 if len(parts)!=2{return "",false}
 values:=strings.Split(parts[1],"_")
 if len(values)!=2||len(values[0])!=14||len(values[1])!=1{return "",false}
 i:=int(values[1][0]-'0')
 suggestionsByToken.Lock()
 s,ok:=suggestionsByToken.items[values[0]]
 suggestionsByToken.Unlock()
 if !ok||time.Now().After(s.Expires)||s.UserID!=userID||s.ChatID!=chatID||i<0||i>=len(s.Titles){return "",false}
 return s.Titles[i].Query,true
}

func suggestionButtons(token string,titles []suggestedTitle)[][]gotgbot.InlineKeyboardButton{
 keyboard:=make([][]gotgbot.InlineKeyboardButton,0,len(titles))
 for i,title:=range titles{
  label:="🎬 "+title.Label
  if len([]rune(label))>60{label=string([]rune(label)[:59])+"…"}
  keyboard=append(keyboard,[]gotgbot.InlineKeyboardButton{{Text:label,CallbackData:"sug|"+token+"_"+string(rune('0'+i))}})
 }
 return keyboard
}
