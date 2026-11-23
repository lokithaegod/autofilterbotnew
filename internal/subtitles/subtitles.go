// Package subtitles contains pure parsing and normalization for Malayalam subtitle files.
package subtitles

import (
 "path/filepath"
 "regexp"
 "sort"
 "strings"
)

var yearPattern = regexp.MustCompile(`(?i)^(.+?)[ ._\-(\[]+((?:19|20)[0-9]{2})(?:[^0-9]|$)`)
var queryYear = regexp.MustCompile(`(?i)^(.*?)[ ._\-(\[]+((?:19|20)[0-9]{2})(?:[^0-9]|$)`)
var words = regexp.MustCompile(`[^a-z0-9]+`)
var fpsPattern=regexp.MustCompile(`(?i)(?:^|[^\d])((?:23[.]976|24[.]000|25[.]000|29[.]970|30[.]000|23|24|25|30|60))\s*fps(?:[^a-z]|$)`)
var trailingGroup=regexp.MustCompile(`(?i)-([a-z][a-z0-9]{1,19})// Package subtitles contains pure parsing and normalization for Malayalam subtitle files.
package subtitles

import (
 "path/filepath"
 "regexp"
 "sort"
 "strings"
)

var yearPattern = regexp.MustCompile(`(?i)^(.+?)[ ._\-(\[]+((?:19|20)[0-9]{2})(?:[^0-9]|$)`)
var queryYear = regexp.MustCompile(`(?i)^(.*?)[ ._\-(\[]+((?:19|20)[0-9]{2})(?:[^0-9]|$)`)
)

type Entry struct {
 UniqueID string `bson:"_id"`
 FileID string `bson:"file_id"`
 FileName string `bson:"file_name"`
 Title string `bson:"title"`
 TitleKey string `bson:"title_key"`
 Year string `bson:"year"`
 Size int64 `bson:"file_size"`
 ChatID int64 `bson:"chat_id"`
 MessageID int64 `bson:"message_id"`
 ReleaseType string `bson:"release_type,omitempty"`
 Resolution string `bson:"resolution,omitempty"`
 ReleaseGroup string `bson:"release_group,omitempty"`
 FPS string `bson:"fps,omitempty"`
}

func Supported(filename string) bool {
 switch strings.ToLower(filepath.Ext(filename)) {
 case ".srt", ".ass", ".vtt": return true
 default: return false
 }
}

func Normalize(s string) string {
 s=strings.ToLower(strings.TrimSpace(s))
 s=strings.Join(strings.Fields(words.ReplaceAllString(s," "))," ")
 return s
}

// Parse requires an explicit release year to avoid attaching subtitles to the wrong movie.
func Parse(name string) (title,year,key string,ok bool) {
 if !Supported(name) {return "","","",false}
 stem:=strings.TrimSuffix(name,filepath.Ext(name))
 m:=yearPattern.FindStringSubmatch(stem)
 if len(m)!=3 {return "","","",false}
 raw:=strings.TrimSpace(strings.Trim(m[1]," ._-[]()"))
 title=strings.Join(strings.Fields(strings.NewReplacer("."," ","_"," ").Replace(raw))," ")
 key=Normalize(title)
 if key=="" {return "","","",false}
 return title,m[2],key,true
}

func SearchTerms(query string)(key,year string){
 query=strings.TrimSpace(query)
 if m:=queryYear.FindStringSubmatch(query);len(m)==3{
  return Normalize(m[1]),m[2]
 }
 return Normalize(query),""
}

 
// Release contains filename-derived video/subtitle metadata; fields may be unknown.
type Release struct {Type,Resolution,Group,FPS string}
type Candidate struct {Entry *Entry; Level string; Score int; Reason string}

// ReleaseFromFilename recognizes explicit release tags without guessing.
func ReleaseFromFilename(name string) Release {
 var out Release
 stem:=strings.TrimSuffix(name,filepath.Ext(name))
 normalized:=Normalize(stem)
 tokens:=strings.Fields(normalized)
 has:=func(s string)bool{for _,t:=range tokens{if t==s{return true}};return false}
 switch {
 case strings.Contains(normalized,"web dl")||strings.Contains(normalized,"webdl")||strings.Contains(normalized,"web download"):out.Type="WEB-DL"
 case strings.Contains(normalized,"blu ray")||strings.Contains(normalized,"bluray")||strings.Contains(normalized,"bdrip")||strings.Contains(normalized,"brrip")||strings.Contains(normalized,"bdremux")||has("remux"):out.Type="BluRay"
 case strings.Contains(normalized,"web rip")||strings.Contains(normalized,"webrip"):out.Type="WEBRip"
 case has("hdtv")||has("pdtv"):out.Type="HDTV"
 case has("dvdrip")||has("dvd"):out.Type="DVDRip"
 }
 for _,p:=range []string{"2160p","1080p","720p","480p"}{if has(p){out.Resolution=p;break}}
 if m:=fpsPattern.FindStringSubmatch(strings.ToLower(stem));len(m)==2{out.FPS=m[1]}
 if m:=trailingGroup.FindStringSubmatch(stem);len(m)==2 {
  tag:=strings.ToLower(m[1])
  if tag!="malayalam"&&tag!="sub"&&tag!="subs"&&tag!="subtitle"&&tag!="srt"&&tag!="ass"&&tag!="vtt"{out.Group=tag}
 }
 return out
}

// Compare is a heuristic: a likely match is not proof that subtitle timing is synchronized.
func Compare(video,sub Release)(level string,score int,reason string){
 var matched,conflicts []string
 known:=0
 check:=func(field,a,b string,weight int){
  if a==""||b==""{return}
  known++
  if strings.EqualFold(a,b){score+=weight;matched=append(matched,field+" matches")}else{score-=weight;conflicts=append(conflicts,field+" differs")}
 }
 check("source",video.Type,sub.Type,4)
 check("release group",video.Group,sub.Group,3)
 check("FPS",video.FPS,sub.FPS,3)
 check("resolution",video.Resolution,sub.Resolution,1)
 if len(conflicts)>0{return "different",score,strings.Join(conflicts,", ")}
 if known==0{return "unknown",score,"Release details unavailable"}
 if score>=4{return "likely",score,strings.Join(matched,", ")}
 return "possible",score,strings.Join(matched,", ")
}

func releasePriority(s string)int{switch s{case "likely":return 3;case "possible":return 2;case "unknown":return 1;default:return 0}}

// Rank evaluates all candidate subtitles against all video releases, favoring
// compatible evidence, but retains conflicting versions with a clear warning.
func Rank(entries []*Entry,videos []string)[]Candidate{
 out:=make([]Candidate,0,len(entries))
 for _,entry:=range entries{
  sub:=ReleaseFromFilename(entry.FileName)
  if entry.ReleaseType!=""{sub.Type=entry.ReleaseType}
  if entry.Resolution!=""{sub.Resolution=entry.Resolution}
  if entry.ReleaseGroup!=""{sub.Group=entry.ReleaseGroup}
  if entry.FPS!=""{sub.FPS=entry.FPS}
  best:=Candidate{Entry:entry,Level:"unknown",Reason:"No movie release information available"}
  for n,name:=range videos{
   level,score,reason:=Compare(ReleaseFromFilename(name),sub)
   next:=Candidate{Entry:entry,Level:level,Score:score,Reason:reason}
   if n==0||releasePriority(next.Level)>releasePriority(best.Level)||(releasePriority(next.Level)==releasePriority(best.Level)&&next.Score>best.Score){best=next}
  }
  out=append(out,best)
 }
 sort.SliceStable(out,func(i,j int)bool{
  if releasePriority(out[i].Level)!=releasePriority(out[j].Level){return releasePriority(out[i].Level)>releasePriority(out[j].Level)}
  if out[i].Score!=out[j].Score{return out[i].Score>out[j].Score}
  return out[i].Entry.FileName<out[j].Entry.FileName
 })
 return out
}
