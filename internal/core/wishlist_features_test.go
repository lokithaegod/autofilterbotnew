package core

import (
 "strings"
 "testing"

 "github.com/Jisin0/autofilterbot/internal/database/mongo"
 "github.com/Jisin0/autofilterbot/internal/model"
)

func TestKeywordFilterParsingAndPrecedence(t *testing.T){
 k,r,ok:=keywordParts("/addfilter download help | Please read the instructions")
 if !ok||k!="download help"||r!="Please read the instructions"{t.Fatalf("unexpected parse: %q %q %v",k,r,ok)}
 for _,invalid:=range []string{"/addfilter","/addfilter missing pipe","/addfilter | empty","/addfilter x | "} {
  if _,_,ok:=keywordParts(invalid);ok{t.Fatalf("accepted invalid input: %q",invalid)}
 }
 local:=[]mongo.KeywordReply{{Keyword:"movie",Reply:"local movie"},{Keyword:"movie request",Reply:"local request"}}
 global:=[]mongo.KeywordReply{{Keyword:"movie request",Reply:"global request"}}
 if got,ok:=matchReply("Movie request please",local,global);!ok||got!="local request"{t.Fatalf("local longest keyword should win: %q %v",got,ok)}
 if got,ok:=matchReply("MOVIE",nil,global);ok||got!=""{t.Fatalf("unexpected global match: %q",got)}
 if got,ok:=matchReply("movie requests",nil,[]mongo.KeywordReply{{Keyword:"movie request",Reply:"bad"}});ok||got!=""{t.Fatalf("word boundary mismatch %q",got)}
 if !keywordMatch("How to download?", "download"){t.Fatal("missing literal keyword match")}
 if keywordMatch("redownload","download"){t.Fatal("partial word should not match")}
 if keywordMatch("abc",".*"){t.Fatal("regex characters must not act as wildcards")}
}

func TestDuplicateReportExactMatchesOnly(t *testing.T){
 files:=[]model.File{
  {UniqueId:"a",FileId:"telegram-id-1",FileName:"Film.2020.720p.mkv",FileSize:1000},
  {UniqueId:"b",FileId:"telegram-id-2",FileName:"Film_2020_720p.mkv",FileSize:1000},
  {UniqueId:"c",FileId:"telegram-id-3",FileName:"Film.2020.1080p.mkv",FileSize:1000},
  {UniqueId:"d",FileId:"telegram-id-1",FileName:"Different release",FileSize:5000},
  {UniqueId:"e",FileId:"telegram-id-4",FileName:"Film.2020.720p.mkv",FileSize:1050},
 }
 pairs:=detectDuplicatePairs(files,10)
 if len(pairs)!=2{t.Fatalf("want two exact-evidence pairs, got %+v",pairs)}
 reasons:=map[string]bool{}
 for _,p:=range pairs{reasons[p.Reason]=true}
 if !reasons["same Telegram file ID"]||!reasons["same filename and exact size"]{t.Fatalf("missing duplicate evidence: %+v",pairs)}
 if len(detectDuplicatePairs(files,1))!=1{t.Fatal("scan must cap results")}
}
func TestDownloadHelpContent(t *testing.T){
 if !strings.Contains(downloadHelpText,"ForceSub")||!strings.Contains(downloadHelpText,"Malayalam") {t.Fatal("download guide missing essential steps")}
 if btn:=downloadHelpButton();len(btn)!=1||btn[0].CallbackData!="downloadhelp"{t.Fatal("invalid help callback")}
}
