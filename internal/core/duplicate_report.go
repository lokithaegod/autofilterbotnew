package core

import (
 "fmt"
 "sort"
 "strings"

 "github.com/Jisin0/autofilterbot/internal/model"
 "github.com/PaulSonOfLars/gotgbot/v2"
 "github.com/PaulSonOfLars/gotgbot/v2/ext"
)

// Duplicates are report-only. A file with a different quality, size or ID
// is never silently deleted or automatically merged.
type duplicatePair struct{A,B model.File;Reason string}
func normalizeDuplicateName(name string)string{
 name=strings.ToLower(strings.TrimSpace(name))
 return strings.Join(strings.Fields(strings.NewReplacer("."," ","_"," ","-"," ").Replace(name))," ")
}
func detectDuplicatePairs(files []model.File,capPairs int)[]duplicatePair{
 byFileID:=map[string]model.File{}
 byNameSize:=map[string]model.File{}
 result:=make([]duplicatePair,0)
 seen:=map[string]bool{}
 for _,f:=range files{
  if f.UniqueId==""{continue}
  matches:=[]struct{other model.File;reason string}{}
  if f.FileId!=""{
   if old,ok:=byFileID[f.FileId];ok&&old.UniqueId!=f.UniqueId{matches=append(matches,struct{other model.File;reason string}{old,"same Telegram file ID"})}
   byFileID[f.FileId]=f
  }
  if n:=normalizeDuplicateName(f.FileName);n!=""&&f.FileSize>0{
   key:=fmt.Sprintf("%s|%d",n,f.FileSize)
   if old,ok:=byNameSize[key];ok&&old.UniqueId!=f.UniqueId{matches=append(matches,struct{other model.File;reason string}{old,"same filename and exact size"})}
   byNameSize[key]=f
  }
  for _,m:=range matches{
   ids:=[]string{f.UniqueId,m.other.UniqueId};sort.Strings(ids)
   key:=ids[0]+"|"+ids[1]
   if seen[key]{continue};seen[key]=true
   result=append(result,duplicatePair{A:m.other,B:f,Reason:m.reason})
   if len(result)>=capPairs{return result}
  }
 }
 return result
}
func DuplicateFileReport(bot *gotgbot.Bot,ctx *ext.Context)error{
 if !privateAdminOnly(bot,ctx){return nil}
 files,truncated,err:=_app.DB.DuplicateCandidates(1500)
 if err!=nil{return err}
 pairs:=detectDuplicatePairs(files,15)
 var b strings.Builder
 fmt.Fprintf(&b,"🔎 Read-only duplicate report\nScanned: %d indexed files\n",len(files))
 if truncated{b.WriteString("⚠️ Bounded sample: there may be more files outside this scan.\n")}
 if len(pairs)==0{b.WriteString("✅ No exact-ID or exact-size/name duplicates in sampled files.")}
 for i,p:=range pairs{
  fmt.Fprintf(&b,"\n%d. %s\n%s\nIDs: %s / %s\n",i+1,p.Reason,p.A.FileName,p.A.UniqueId,p.B.UniqueId)
  if b.Len()>3500{b.WriteString("\n…report shortened");break}
 }
 b.WriteString("\nNo files were deleted or modified.")
 _,err=ctx.Message.Reply(bot,b.String(),nil)
 return err
}
