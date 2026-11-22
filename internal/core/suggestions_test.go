package core

import (
 "strings"
 "testing"
 "time"
)

func TestIndexedTitleExtraction(t *testing.T){
 files:=[]string{
  "Bruce.Almighty.2003.720p.BrRip.x264.YIFY.mp4",
  "Bruce.Almighty.2003.1080p.BluRay.mkv",
  "Interstellar.2014.1080p.BluRay.mkv",
  "Once.Upon.A.TimeIn.Hollywood.2019.1080p.mkv",
  "Ted.Lasso.S01E01.720p.mkv",
  "Ted.Lasso.S01E02.1080p.mkv",
  "fri.s05e06.br720p.299mb-PaHe.in.mkv",
  "Movie.tamil.srt",
 }
 titles:=uniqueTitlesFromFilenames(files)
 if len(titles)!=5{t.Fatalf("expected 5 title groups, got %d: %+v",len(titles),titles)}
 for _,tt:=range titles{
  if strings.Contains(strings.ToLower(tt.Label),"720p")||strings.Contains(strings.ToLower(tt.Label),"mkv"){
   t.Errorf("release metadata leaked into title: %q",tt.Label)
  }
 }
}

func TestSuggestionRankingTypoAndPartial(t *testing.T){
 files:=[]string{"Bruce.Almighty.2003.720p.mkv","Interstellar.2014.1080p.mp4",
  "Once.Upon.A.TimeIn.Hollywood.2019.720p.mkv","Ted.Lasso.S01E01.mkv",
  "The.100.S01E01.mkv","Friday.2025.720p.mkv","Prometheus.2012.720p.mkv"}
 titles:=uniqueTitlesFromFilenames(files)
 cases:=[]struct{query,expected string}{
  {"bruse almighty","Bruce Almighty (2003)"},
  {"intersteller","Interstellar (2014)"},
  {"ted laso","Ted Lasso"},
  {"once upon hollywood","Once Upon A TimeIn Hollywood (2019)"},
  {"promethus","Prometheus (2012)"},
 }
 for _,tt:=range cases{
  out:=rankSuggestions(tt.query,titles)
  if len(out)==0||out[0].Label!=tt.expected{t.Errorf("%q => %+v, expected %q",tt.query,out,tt.expected)}
  if len(out)>3{t.Errorf("too many: %+v",out)}
 }
 for _,query:=range []string{"zxqvvv","abcdefghijk","bruce infinity"}{
  if got:=rankSuggestions(query,titles);len(got)>0{t.Errorf("unrelated query %q => %+v",query,got)}
 }
}

func TestSuggestionCallbacksAndAuthorization(t *testing.T){
 titles:=[]suggestedTitle{{Label:"Bruce Almighty (2003)",Query:"Bruce Almighty 2003",Normal:"bruce almighty"}}
 token:=saveSuggestionSession(9001,-10021,titles)
 keyboard:=suggestionButtons(token,titles)
 if len(keyboard)!=1||len(keyboard[0][0].CallbackData)>64{t.Fatalf("invalid markup: %+v",keyboard)}
 query,ok:=suggestionFromCallback(keyboard[0][0].CallbackData,9001,-10021)
 if !ok||query!="Bruce Almighty 2003"{t.Fatalf("click didn't resolve: %q %v",query,ok)}
 if _,ok:=suggestionFromCallback(keyboard[0][0].CallbackData,9002,-10021);ok{t.Fatal("other user can hijack suggestions")}
 if _,ok:=suggestionFromCallback(keyboard[0][0].CallbackData,9001,-222);ok{t.Fatal("other chat can hijack suggestions")}
 if _,ok:=suggestionFromCallback("sug|"+token+"_5",9001,-10021);ok{t.Fatal("invalid suggestion position")}
 suggestionsByToken.Lock()
 old:=suggestionsByToken.items[token]
 old.Expires=time.Now().Add(-time.Second)
 suggestionsByToken.items[token]=old
 suggestionsByToken.Unlock()
 if _,ok:=suggestionFromCallback(keyboard[0][0].CallbackData,9001,-10021);ok{t.Fatal("expired suggestion accepted")}
}
