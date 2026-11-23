package core

import (
 "strings"
 "testing"

 "github.com/Jisin0/autofilterbot/internal/autofilter"
 "github.com/Jisin0/autofilterbot/internal/config"
 "github.com/Jisin0/autofilterbot/internal/model"
)

func testDirectFile(id,name string)autofilter.File{
 return autofilter.File{File:model.File{UniqueId:id,FileName:name,FileSize:1400000000}}
}

func TestDirectFilesMalayalamFirstAndSubtitleLast(t *testing.T){
 pages:=[]autofilter.Files{{
  testDirectFile("en1","Cast.Away.2000.720p.English.Sub.mkv"),
  testDirectFile("ml2","Cast.Away.2000.1080p.BluRay.mkv"),
  testDirectFile("other1","Cast.Away.2000.480p.mkv"),
  testDirectFile("srt","Cast.Away.2000.English.srt"),
  testDirectFile("ml1","Cast.Away.2000.720p.BluRay.mp4"),
  testDirectFile("mlsrt","Cast.Away.2000.720p.srt"),
  testDirectFile("unknown","Cast.Away.2000.1080p.mkv"),
  testDirectFile("malzip","Cast.Away.2000.മലയാളം.zip"),
 }}
 tags:=map[string]bool{"ml1":true,"ml2":true,"mlsrt":true}
 groups:=groupDirectMovieFiles(pages,tags)
 ml:=groups.malayalam()
 if len(ml)!=4{t.Fatalf("expected 4 Malayalam files, got %d",len(ml))}
 if ml[0].UniqueId!="ml1"||ml[1].UniqueId!="ml2"||ml[2].UniqueId!="mlsrt"||ml[3].UniqueId!="malzip"{
  t.Fatalf("Malayalam videos must sort 720p then 1080p and subtitles come last: %+v",ml)
 }
 if len(groups.english())!=2 || groups.english()[0].UniqueId!="en1"||groups.english()[1].UniqueId!="srt"{
  t.Fatalf("explicit English video then SRT expected: %+v",groups.english())
 }
 if len(groups.Other)!=2{t.Fatalf("unlabeled files must remain Other: %+v",groups.Other)}
 if groups.firstKind()!="ml"{t.Fatal("Malayalam section not default")}
 keyboard:=directFileKeyboard("abcdefghijklmno",groups,"ml",0,1,"AutofilterMGbot",&config.Config{})
 if !strings.Contains(keyboard[0][0].Text,"Malayalam Sub & Files (4)") {t.Fatalf("unexpected header: %+v",keyboard)}
 for _,row:=range keyboard{for _,button:=range row{
  if strings.HasPrefix(button.CallbackData,"card|"){t.Fatal("movie card callback must not appear in direct view")}
  if len(button.CallbackData)>64{t.Fatalf("callback over Telegram limit: %q",button.CallbackData)}
 }}
}

func TestDirectEnglishRequiresPositiveEvidence(t *testing.T){
 examples:=[]struct{name string;english bool}{
  {"Film.2018.ENG.1080p.BluRay.mkv",false},
  {"Film.2018.English.Audio.mkv",false},
  {"Film.2018.720p.English.Sub.mp4",true},
  {"Film.2018.1080p.Eng.Subs.mkv",true},
  {"Film.2018.English.srt",true},
  {"Film.2018.Eng.srt",true},
  {"Film.2018.Tamil.srt",false},
  {"Film.2018.BluRay.മലയാളം.srt",false},
  {"Film.2018.English.zip",true},
 }
 for _,tc:=range examples{if got:=isEnglishMarked(tc.name);got!=tc.english{t.Errorf("%q English=%v want %v",tc.name,got,tc.english)}}
}

func TestDirectNoMalayalamOrEmptyCategories(t *testing.T){
 groups:=groupDirectMovieFiles([]autofilter.Files{{testDirectFile("one","Movie.2020.1080p.mkv")}},nil)
 if groups.firstKind()!="other"{t.Fatalf("expected Other first if no tags, got %q",groups.firstKind())}
 keyboard:=directFileKeyboard("abcdefghijklmno",groups,groups.firstKind(),0,1,"AutofilterMGbot",&config.Config{})
 for _,row:=range keyboard{for _,button:=range row{
  if strings.Contains(button.Text,"English Sub Files")||strings.Contains(button.Text,"Malayalam Sub & Files"){
   t.Fatalf("empty language section should be hidden: %+v",keyboard)
  }
 }}
}

func TestDirectPagingAndSubtitleLastAcrossPages(t *testing.T){
 tagged:=map[string]bool{}
 files:=make(autofilter.Files,0)
 for i:=0;i<9;i++{
  id:=string(rune('a'+i))
  files=append(files,testDirectFile(id,"Film.2020.720p.mkv"))
  tagged[id]=true
 }
 files=append(files,testDirectFile("subtitle","Film.2020.മലയാളം.srt"))
 groups:=groupDirectMovieFiles([]autofilter.Files{files},tagged)
 if got:=len(groups.malayalam());got!=10{t.Fatalf("expected ten, got %d",got)}
 key:=directFileKeyboard("abcdefghijklmno",groups,"ml",0,1,"AutofilterMGbot",&config.Config{})
 next:=false
 for _,row:=range key{for _,button:=range row{if button.CallbackData=="direct|abcdefghijklmno_ml_1"{next=true}}}
 if !next{t.Fatal("next page missing")}
 second:=directFileKeyboard("abcdefghijklmno",groups,"ml",1,1,"AutofilterMGbot",&config.Config{})
 foundSRT:=false
 for _,row:=range second{for _,button:=range row{if strings.Contains(button.Text,"മലയാളം.srt"){foundSRT=true}}}
 if !foundSRT{t.Fatal("Malayalam subtitle should be on final page after video files")}
}
