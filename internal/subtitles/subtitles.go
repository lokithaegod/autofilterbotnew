// Package subtitles contains pure parsing and normalization for Malayalam subtitle files.
package subtitles

import (
 "path/filepath"
 "regexp"
 "strings"
)

var yearPattern = regexp.MustCompile(`(?i)^(.+?)[ ._\-(\[]+((?:19|20)[0-9]{2})(?:[^0-9]|$)`)
var queryYear = regexp.MustCompile(`(?i)^(.*?)[ ._\-(\[]+((?:19|20)[0-9]{2})(?:[^0-9]|$)`)
var words = regexp.MustCompile(`[^a-z0-9]+`)

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
