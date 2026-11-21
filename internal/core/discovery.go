package core

import (
 "encoding/json"
 "fmt"
 "html"
 "net/http"
 "io"
 "regexp"
 "sort"
 "net/url"
 "os"
 "strconv"
 "strings"
 "sync"
 "time"

 "github.com/Jisin0/autofilterbot/internal/autofilter"
 "github.com/Jisin0/autofilterbot/internal/database/mongo"
 "github.com/Jisin0/autofilterbot/internal/model"
 "github.com/PaulSonOfLars/gotgbot/v2"
 "github.com/PaulSonOfLars/gotgbot/v2/ext"
)

var searchThrottle = struct {
 sync.Mutex
 hits map[int64][]time.Time
}{hits:make(map[int64][]time.Time)}

// SearchAllowed prevents repeated high-frequency searches without blocking admins.
func SearchAllowed(userID int64) bool {
 if containsI64(_app.Admins,userID){return true}
 searchThrottle.Lock();defer searchThrottle.Unlock()
 now:=time.Now()
 hits:=searchThrottle.hits[userID][:0]
 for _,v:=range searchThrottle.hits[userID] {if now.Sub(v)<15*time.Second {hits=append(hits,v)}}
 if len(hits)>=6 {searchThrottle.hits[userID]=hits;return false}
 searchThrottle.hits[userID]=append(hits,now)
 if len(searchThrottle.hits)>1000 {
  for id,v:=range searchThrottle.hits {if len(v)==0||now.Sub(v[len(v)-1])>time.Minute {delete(searchThrottle.hits,id)}}
 }
 return true
}

func CheckSearchAccess(bot *gotgbot.Bot,ctx *ext.Context) bool {
 if ctx.Message==nil||ctx.Message.From==nil{return false}
 id:=ctx.Message.From.Id
 blocked,err:=_app.DB.IsUserBlocked(id)
 if err!=nil {
  _,_=ctx.Message.Reply(bot,"⚠️ Search temporarily unavailable. Please try again.",nil)
  return false
 }
 if blocked {
  _,_=ctx.Message.Reply(bot,"Your access to this bot is restricted. Contact an admin.",nil)
  return false
 }
 if !SearchAllowed(id){
  _,_=ctx.Message.Reply(bot,"⏳ Too many searches. Try again shortly.",nil)
  return false
 }
 return true
}

func parseDiscover(input string)(mongo.CatalogFilter,error){
 f:=mongo.CatalogFilter{Sort:"newest"}
 parts:=strings.Fields(input)
 var words []string
 for _,p:=range parts {
  pair:=strings.SplitN(p,":",2)
  if len(pair)!=2 {words=append(words,p);continue}
  v:=strings.ToLower(pair[1])
  switch strings.ToLower(pair[0]) {
  case "year":
   n,e:=strconv.Atoi(v);if e!=nil||n<1888||n>2100{return f,fmt.Errorf("invalid year")}
   f.Year=v
  case "quality":
   valid:=map[string]bool{"480p":true,"720p":true,"1080p":true,"2160p":true,"4k":true}
   if !valid[v]{return f,fmt.Errorf("invalid quality")}
   if v=="4k"{v="2160p"};f.Quality=v
  case "lang","language":
   aliases:=map[string]string{"malayalam":"mal","tamil":"tam","hindi":"hin","english":"eng","mal":"mal","tam":"tam","hin":"hin","eng":"eng"}
   var ok bool;f.Language,ok=aliases[v];if !ok{return f,fmt.Errorf("invalid language")}
  case "type":
   if v!="video"&&v!="document"&&v!="audio"{return f,fmt.Errorf("invalid file type")}
   f.Type=v
  case "sort":
   if v!="newest"&&v!="smallest"&&v!="largest"{return f,fmt.Errorf("invalid sort")}
   f.Sort=v
  case "minmb","maxmb":
   amount,e:=strconv.ParseInt(v,10,64);if e!=nil||amount<0||amount>100000{return f,fmt.Errorf("invalid size")}
   if pair[0]=="minmb"{f.MinSize=amount*1024*1024}else{f.MaxSize=amount*1024*1024}
  default:words=append(words,p)
  }
 }
 if f.MaxSize>0&&f.MinSize>f.MaxSize{return f,fmt.Errorf("minmb greater than maxmb")}
 f.Query=strings.Join(words," ")
 if len(f.Query)>100{return f,fmt.Errorf("search too long")}
 return f,nil
}

func sendCatalog(bot *gotgbot.Bot,chatID int64,category string,files []model.File) error {
 if len(files)==0 {_,err:=bot.SendMessage(chatID,"No matching indexed files found.",nil);return err}
 if len(files)>15 {files=files[:15]}
 wrapped:=make(autofilter.Files,0,len(files))
 for _,f:=range files {wrapped=append(wrapped,autofilter.File{File:f})}
 keyboard:=wrapped.Process(chatID,bot.Username,_app.Config)
 _,err:=bot.SendMessage(chatID,fmt.Sprintf("📁 %s • %d results (tap to receive)",html.EscapeString(category),len(files)),&gotgbot.SendMessageOpts{ParseMode:gotgbot.ParseModeHTML,ReplyMarkup:gotgbot.InlineKeyboardMarkup{InlineKeyboard:keyboard}})
 return err
}

func AdvancedFind(bot *gotgbot.Bot,ctx *ext.Context)error{
 if ctx.Message==nil{return nil}
 if !CheckSearchAccess(bot,ctx){return nil}
 text:=strings.TrimSpace(strings.TrimPrefix(ctx.Message.Text,strings.Fields(ctx.Message.Text)[0]))
 if text=="" {_,e:=ctx.Message.Reply(bot,"Usage: /find Bruce Almighty year:2003 quality:720p lang:eng sort:smallest\nOptions: type:video minmb:100 maxmb:2000",nil);return e}
 filter,err:=parseDiscover(text)
 if err!=nil{_,e:=ctx.Message.Reply(bot,"Invalid filter: "+err.Error(),nil);return e}
 files,err:=_app.DB.CatalogFiles(filter,40)
 if err!=nil{return err}
 if filter.Query!="" {_= _app.DB.LogSearch(filter.Query)}
 return sendCatalog(bot,ctx.Message.Chat.Id,"Filtered files",files)
}

func BrowseFiles(bot *gotgbot.Bot,ctx *ext.Context)error{
 keyboard:=gotgbot.InlineKeyboardMarkup{InlineKeyboard:[][]gotgbot.InlineKeyboardButton{
  {{Text:"🎞 Movies",CallbackData:"browse|movies"},{Text:"📺 Series",CallbackData:"browse|series"}},
  {{Text:"📄 Documents",CallbackData:"browse|documents"},{Text:"🆕 Latest uploads",CallbackData:"browse|latest"}},
  {{Text:"🌐 Malayalam",CallbackData:"browse|mal"},{Text:"🌐 Tamil",CallbackData:"browse|tam"}},
  {{Text:"🌐 Hindi",CallbackData:"browse|hin"},{Text:"🌐 English",CallbackData:"browse|eng"}},
 }}
 _,err:=bot.SendMessage(ctx.EffectiveChat.Id,"📚 Browse indexed files by category or language", &gotgbot.SendMessageOpts{ReplyMarkup:keyboard})
 return err
}

func BrowseCallback(bot *gotgbot.Bot,ctx *ext.Context)error{
 c:=ctx.CallbackQuery
 cat:=strings.TrimPrefix(c.Data,"browse|")
 allowed:=map[string]bool{"movies":true,"series":true,"documents":true,"latest":true,"mal":true,"tam":true,"hin":true,"eng":true}
 if !allowed[cat]{_,_=c.Answer(bot,&gotgbot.AnswerCallbackQueryOpts{Text:"Unknown category",ShowAlert:true});return nil}
 // Verify rate limits and restrictions before catalog browsing.
 blocked,err:=_app.DB.IsUserBlocked(c.From.Id)
 if err!=nil||blocked {_,_=c.Answer(bot,&gotgbot.AnswerCallbackQueryOpts{Text:"Access unavailable",ShowAlert:true});return nil}
 if !SearchAllowed(c.From.Id){_,_=c.Answer(bot,&gotgbot.AnswerCallbackQueryOpts{Text:"Try again shortly",ShowAlert:true});return nil}
 f:=mongo.CatalogFilter{Category:cat}
 if len(cat)==3 {f.Category="";f.Language=cat}
 files,err:=_app.DB.CatalogFiles(f,40)
 if err!=nil{_,_=c.Answer(bot,&gotgbot.AnswerCallbackQueryOpts{Text:"Database unavailable",ShowAlert:true});return nil}
 _,_=c.Answer(bot,nil)
 return sendCatalog(bot,c.From.Id,strings.ToUpper(cat),files)
}

func AdminDashboard(bot *gotgbot.Bot,ctx *ext.Context)error{
 if !_app.AuthAdmin(ctx){return nil}
 stats,err:=_app.DB.Stats()
 if err!=nil{return err}
 top,err:=_app.DB.TopSearches(8)
 if err!=nil{return err}
 text:=fmt.Sprintf("📊 Admin Dashboard\nUsers: %d\nGroups: %d\nFiles: %s\n\nPopular searches:",stats.Users,stats.Groups,stats.Files)
 if len(top)==0{text+="\nNo searches recorded yet."}
 for i,t:=range top {text+=fmt.Sprintf("\n%d. %s — %d",i+1,t.Query,t.Count)}
 _,err=ctx.Message.Reply(bot,text,nil)
 return err
}

func SetBan(bot *gotgbot.Bot,ctx *ext.Context)error{
 if !_app.AuthAdmin(ctx){return nil}
 parts:=strings.Fields(ctx.Message.Text)
 if len(parts)!=2 {_,err:=ctx.Message.Reply(bot,"Usage: /banuser 123456789 or /unbanuser 123456789",nil);return err}
 id,err:=strconv.ParseInt(parts[1],10,64)
 if err!=nil||id<=0 {_,e:=ctx.Message.Reply(bot,"Invalid numeric user ID.",nil);return e}
 if containsI64(_app.Admins,id) {_,e:=ctx.Message.Reply(bot,"Cannot block a bot administrator.",nil);return e}
 blocked:=strings.HasPrefix(parts[0],"/banuser")
 err=_app.DB.SetUserBlocked(id,blocked)
 if err!=nil{return err}
 _,err=ctx.Message.Reply(bot,fmt.Sprintf("User %d blocked: %t",id,blocked),nil)
 return err
}

type movieInfo struct {
 Title string `json:"Title"`
 Year string `json:"Year"`
 Genre string `json:"Genre"`
 Plot string `json:"Plot"`
 IMDBRating string `json:"imdbRating"`
 Poster string `json:"Poster"`
 Response string `json:"Response"`
 Error string `json:"Error"`
}
func MovieInfo(bot *gotgbot.Bot,ctx *ext.Context)error{
 if ctx.Message==nil{return nil}
 key:=os.Getenv("OMDB_API_KEY")
 if key=="" {_,err:=ctx.Message.Reply(bot,"Movie cards require an OMDb API key. Admin: add OMDB_API_KEY in Render Environment.",nil);return err}
 words:=strings.Fields(ctx.Message.Text)
 if len(words)<2{_,err:=ctx.Message.Reply(bot,"Usage: /movie Bruce Almighty",nil);return err}
 name:=strings.Join(words[1:]," ")
 if len(name)>90 {_,err:=ctx.Message.Reply(bot,"Title is too long.",nil);return err}
 client:=&http.Client{Timeout:7*time.Second}
 endpoint:="https://www.omdbapi.com/?apikey="+url.QueryEscape(key)+"&t="+url.QueryEscape(name)+"&plot=short"
 resp,err:=client.Get(endpoint)
 if err!=nil{_,e:=ctx.Message.Reply(bot,"Movie information service unavailable.",nil);return e}
 defer resp.Body.Close()
 if resp.StatusCode!=200 {_,e:=ctx.Message.Reply(bot,"Movie information service returned an error.",nil);return e}
 var info movieInfo
 if err=json.NewDecoder(io.LimitReader(resp.Body,20000)).Decode(&info);err!=nil{return err}
 if info.Response!="True" {_,e:=ctx.Message.Reply(bot,"No matching movie metadata found.",nil);return e}
 msg:=fmt.Sprintf("🎬 %s (%s)\nGenre: %s\nIMDb: %s\n\n%s",info.Title,info.Year,info.Genre,info.IMDBRating,info.Plot)
 if strings.HasPrefix(info.Poster,"https://"){msg+="\nPoster: "+info.Poster}
 _,err=ctx.Message.Reply(bot,msg,nil)
 return err
}

func fuzzyDistance(a,b string) int {
 x:=[]rune(a);y:=[]rune(b)
 prev:=make([]int,len(y)+1)
 for i:=range prev{prev[i]=i}
 for i,c:=range x {
  current:=make([]int,len(y)+1);current[0]=i+1
  for j,d:=range y {
   cost:=0;if c!=d{cost=1}
   ins:=current[j]+1;del:=prev[j+1]+1;sub:=prev[j]+cost
   n:=ins;if del<n{n=del};if sub<n{n=sub}
   current[j+1]=n
  }
  prev=current
 }
 return prev[len(y)]
}
var normalizeSuggest = regexp.MustCompile(`[^a-z0-9 ]+`)
var removeRelease = regexp.MustCompile(`\b(19|20)[0-9]{2}\b`)
func normalizeTitle(s string)string{
 s=strings.ToLower(s)
 s=removeRelease.ReplaceAllString(s," ")
 s=normalizeSuggest.ReplaceAllString(s," ")
 return strings.Join(strings.Fields(s)," ")
}

func SuggestTitles(query string) ([]string,error) {
 candidates,err:=_app.DB.CatalogFiles(mongo.CatalogFilter{Sort:"newest"},100)
 if err!=nil{return nil,err}
 q:=normalizeTitle(query)
 if q==""{return nil,nil}
 type match struct{text string;score int}
 matches:=make([]match,0,10)
 seen:=map[string]bool{}
 for _,file:=range candidates {
  normalized:=normalizeTitle(file.FileName)
  words:=strings.Fields(normalized)
  // Compare to the title prefix rather than full codec, rip, and file extension.
  if len(words)>6{words=words[:6]}
  title:=strings.Join(words," ")
  if title==""||seen[title]{continue}
  seen[title]=true
  score:=fuzzyDistance(q,title)
  if strings.Contains(title,q)||strings.Contains(q,title){score=0}
  // Accept reasonably close matches, not unrelated files.
  tolerance:=len([]rune(q))/3+2
  if score<=tolerance {matches=append(matches,match{title,score})}
 }
 sort.Slice(matches,func(i,j int)bool{return matches[i].score<matches[j].score})
 var out []string
 for i:=0;i<len(matches)&&i<3;i++{out=append(out,matches[i].text)}
 return out,nil
}
