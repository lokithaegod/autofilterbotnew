package core

import (
 "fmt"
 "regexp"
 "sort"
 "strings"
 "unicode/utf8"

 "github.com/Jisin0/autofilterbot/internal/database/mongo"
 "github.com/PaulSonOfLars/gotgbot/v2"
 "github.com/PaulSonOfLars/gotgbot/v2/ext"
)

// All modifications are restricted to configured bot administrators.
// Local filters are chat-scoped; global filters are bot-wide.
func keywordParts(text string)(string,string,bool){
 words:=strings.SplitN(strings.TrimSpace(text)," ",2)
 if len(words)!=2{return "","",false}
 parts:=strings.SplitN(strings.TrimSpace(words[1]),"|",2)
 if len(parts)!=2{return "","",false}
 key:=strings.ToLower(strings.TrimSpace(parts[0]))
 reply:=strings.TrimSpace(parts[1])
 if key==""||reply==""||utf8.RuneCountInString(key)>65||utf8.RuneCountInString(reply)>2500{return "","",false}
 return key,reply,true
}
func filterScope(ctx *ext.Context,global bool)(string,int64,bool){
 if global{return "global",0,true}
 if ctx.EffectiveChat==nil{return "",0,false}
 if ctx.EffectiveChat.Type!="supergroup"&&ctx.EffectiveChat.Type!="group"{return "",0,false}
 return "chat",ctx.EffectiveChat.Id,true
}
func AddKeywordFilter(bot *gotgbot.Bot,ctx *ext.Context)error{
 if !_app.AuthAdmin(ctx){return nil}
 global:=strings.HasPrefix(strings.ToLower(ctx.Message.Text),"/addglobalfilter")
 scope,chatID,ok:=filterScope(ctx,global)
 if !ok{_,err:=ctx.Message.Reply(bot,"Add local filters inside a group. Use /addglobalfilter from any admin chat.",nil);return err}
 key,reply,ok:=keywordParts(ctx.Message.Text)
 if !ok{_,err:=ctx.Message.Reply(bot,"Usage: /addfilter keyword | automatic reply\nOr: /addglobalfilter keyword | automatic reply",nil);return err}
 err:=_app.DB.PutKeywordReply(mongo.KeywordReply{Scope:scope,ChatID:chatID,Keyword:key,Reply:reply})
 if err!=nil{return err}
 _,err=ctx.Message.Reply(bot,fmt.Sprintf("✅ %s keyword filter saved for: %s",scope,key),nil)
 return err
}
func DeleteKeywordFilter(bot *gotgbot.Bot,ctx *ext.Context)error{
 if !_app.AuthAdmin(ctx){return nil}
 global:=strings.HasPrefix(strings.ToLower(ctx.Message.Text),"/delglobalfilter")
 scope,chatID,ok:=filterScope(ctx,global)
 if !ok{_,err:=ctx.Message.Reply(bot,"Local filters must be managed inside their group.",nil);return err}
 args:=ctx.Args();if len(args)<2{_,err:=ctx.Message.Reply(bot,"Usage: /delfilter keyword or /delglobalfilter keyword",nil);return err}
 key:=strings.ToLower(strings.TrimSpace(strings.Join(args[1:]," ")))
 deleted,err:=_app.DB.DeleteKeywordReply(scope,chatID,key);if err!=nil{return err}
 msg:="No matching filter found";if deleted{msg="✅ Filter deleted"}
 _,err=ctx.Message.Reply(bot,msg,nil);return err
}
func ListKeywordFilters(bot *gotgbot.Bot,ctx *ext.Context)error{
 if !_app.AuthAdmin(ctx){return nil}
 global:=strings.HasPrefix(strings.ToLower(ctx.Message.Text),"/globalfilters")
 scope,chatID,ok:=filterScope(ctx,global)
 if !ok{_,err:=ctx.Message.Reply(bot,"Local filters must be viewed inside their group.",nil);return err}
 list,err:=_app.DB.ListKeywordReplies(scope,chatID);if err!=nil{return err}
 msg:="No saved "+scope+" keyword filters."
 if len(list)>0{
  var b strings.Builder
  fmt.Fprintf(&b,"%s keyword filters (%d):\n",scope,len(list))
  for _,f:=range list{if b.Len()+len(f.Keyword)>3600{b.WriteString("\n…more filters omitted");break};b.WriteString("• "+f.Keyword+"\n")}
  msg=b.String()
 }
 _,err=ctx.Message.Reply(bot,msg,nil);return err
}

func keywordMatch(input,keyword string)bool {
 // Boundary-aware, case-insensitive literal matching; no regex injection.
 if keyword==""{return false}
 re:=regexp.MustCompile(`(?i)(^|[^[:alnum:]])`+regexp.QuoteMeta(keyword)+`($|[^[:alnum:]])`)
 return re.MatchString(input)
}
func matchReply(input string,local,global []mongo.KeywordReply)(string,bool){
 // Longest matching local keyword wins; global replies only fill gaps.
 for _,list:=range [][]mongo.KeywordReply{local,global}{
  sort.SliceStable(list,func(i,j int)bool{return len(list[i].Keyword)>len(list[j].Keyword)})
  for _,f:=range list{if keywordMatch(input,f.Keyword){return f.Reply,true}}
 }
 return "",false
}
func handleKeywordReply(bot *gotgbot.Bot,ctx *ext.Context)(bool,error){
 m:=ctx.Message
 if m==nil||m.Text==""||strings.HasPrefix(strings.TrimSpace(m.Text),"/"){return false,nil}
 if m.Chat.Type!="group"&&m.Chat.Type!="supergroup"{return false,nil}
 local,err:=_app.DB.ListKeywordReplies("chat",m.Chat.Id);if err!=nil{return false,err}
 global,err:=_app.DB.ListKeywordReplies("global",0);if err!=nil{return false,err}
 response,ok:=matchReply(m.Text,local,global)
 if !ok{return false,nil}
 _,err=m.Reply(bot,response,nil)
 return true,err
}
