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
 if len(groups.english())!=4 || groups.english()[0].UniqueId!="other1"||groups.english()[1].UniqueId!="en1"||groups.english()[2].UniqueId!="unknown"||groups.english()[3].UniqueId!="srt"{
  t.Fatalf("all non-Tamil regular files should be English, videos ordered before SRT: %+v",groups.english())
 }
 if groups.firstKind()!="ml"{t.Fatal("Malayalam section not default")}
 keyboard:=directFileKeyboard("abcdefghijklmno",groups,"ml",0,1,"AutofilterMGbot",&config.Config{})
 if !strings.Contains(keyboard[0][0].Text,"Malayalam Sub & Files 👇 (4)") {t.Fatalf("unexpected header: %+v",keyboard)}
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
 if groups.firstKind()!="en"{t.Fatalf("expected English when no tags, got %q",groups.firstKind())}
 keyboard:=directFileKeyboard("abcdefghijklmno",groups,groups.firstKind(),0,1,"AutofilterMGbot",&config.Config{})
 for _,row:=range keyboard{for _,button:=range row{
  if strings.Contains(button.Text,"Malayalam Sub & Files")||strings.Contains(button.Text,"Tamil Dubbed Files")||strings.Contains(button.Text,"Unverified"){
   t.Fatalf("empty or removed section should be hidden: %+v",keyboard)
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

func TestTamilDubbedGroupAndVisibleButtons(t *testing.T){
 files:=[]autofilter.Files{{
  testDirectFile("ml","Film.2020.720p.mkv"),
  testDirectFile("ta1","Film.2020.720p.Tamil.Dubbed.mkv"),
  testDirectFile("ta2","Film.2020.1080p.Tam.Dub.mp4"),
  testDirectFile("srt","Film.2020.Tamil.srt"),
  testDirectFile("other","Film.2020.1080p.BluRay.mkv"),
 }}
 groups:=groupDirectMovieFiles(files,map[string]bool{"ml":true})
 if len(groups.TamilVideos)!=2{t.Fatalf("expected 2 Tamil dubbed, got %d",len(groups.TamilVideos))}
 if groups.TamilVideos[0].UniqueId!="ta1"||groups.TamilVideos[1].UniqueId!="ta2"{t.Fatalf("Tamil dubbed files not quality-sorted: %+v",groups.TamilVideos)}
 if len(groups.english())!=2{t.Fatalf("Tamil subtitle and remaining regular video belong in English section: %+v",groups.english())}
 keyboard:=directFileKeyboard("abcdefghijklmno",groups,"ml",0,1,"AutoFilterMGbot",&config.Config{})
 malHeader:=keyboard[0][0].Text
 if !strings.Contains(malHeader,"Malayalam Sub & Files 👇"){t.Errorf("arrow missing: %q",malHeader)}
 hasTamil,hasEnglish:=false,false
 for _,row:=range keyboard{for _,btn:=range row{
  if strings.Contains(btn.Text,"Tamil Dubbed Files (2)"){hasTamil=true}
  if strings.Contains(btn.Text,"English Sub Files (2)"){hasEnglish=true}
  if btn.Text=="📁 Other Files (2)"{t.Error("old Other Files label still shown")}
 }}
 if !hasTamil||!hasEnglish{t.Errorf("missing Tamil or English section: %+v",keyboard)}
}
func TestTamilDubbedDoesNotMatchSubtitlesOrPlainLanguage(t *testing.T){
 for _,name:=range []string{"Film.2020.Tamil.srt","Film.2020.Tamil.Audio.mkv","Film.2020.Tamil.Sub.mkv","Film.2020.Tamil.Subtitles.zip"}{
  if isTamilDubbedVideo(name){t.Errorf("false Tamil dubbed classification: %s",name)}
 }
 for _,name:=range []string{"Film.2020.Tamil.Dubbed.mkv","Film.2020.Tam.Dub.mp4","Film.2020.Tamil-Dub-720p.mkv"}{
  if !isTamilDubbedVideo(name){t.Errorf("missed explicit Tamil dub: %s",name)}
 }
}

func TestThreeGroupsNoOtherAndOwnerPrecedence(t *testing.T){
 pages:=[]autofilter.Files{{
  testDirectFile("tamil","Film.2020.1080p.Tamil.Dubbed.mkv"),
  testDirectFile("both","Film.2020.720p.Tamil.Dubbed.mkv"),
  testDirectFile("regular","Film.2020.720p.ENG.BluRay.mkv"),
  testDirectFile("unknown","Film.2020.1080p.WEBRip.mkv"),
  testDirectFile("engsrt","Film.2020.English.srt"),
  testDirectFile("genericzip","Film.2020.Subtitles.zip"),
  testDirectFile("mal","Film.2020.1080p.mkv"),
  testDirectFile("malsrt","Film.2020.മലയാളം.srt"),
 }}
 groups:=groupDirectMovieFiles(pages,map[string]bool{"both":true,"mal":true})
 if len(groups.malayalam())!=3{t.Fatalf("Malayalam override and SRT expected: %+v",groups.malayalam())}
 if groups.malayalam()[2].UniqueId!="malsrt"{t.Fatal("Malayalam SRT must be last")}
 if len(groups.TamilVideos)!=1||groups.TamilVideos[0].UniqueId!="tamil"{t.Fatalf("Tamil category must exclude Malayalam-approved Tamil file: %+v",groups.TamilVideos)}
 if len(groups.english())!=4{t.Fatalf("regular movies plus subtitle documents must stay accessible: %+v",groups.english())}
 if groups.english()[0].UniqueId!="regular"||groups.english()[1].UniqueId!="unknown"||groups.english()[2].UniqueId!="engsrt"||groups.english()[3].UniqueId!="genericzip"{t.Fatalf("English video order then subtitle documents: %+v",groups.english())}
 keyboard:=directFileKeyboard("abcdefghijklmno",groups,"ml",0,123,"AutoFilterMGbot",&config.Config{})
 for _,row:=range keyboard{for _,button:=range row{
  if strings.Contains(button.Text,"Unverified")||strings.Contains(button.Text,"Other Files"){t.Fatalf("Other/Unverified category still present: %+v",keyboard)}
 }}
 if len(groupDirectMovieFiles(pages,nil).english())!=5{t.Fatal("untagged regular releases should be classified English")}
 // A callback from an old message containing the former other group
 // should transparently show English instead of becoming an empty tab.
 old:=directFileKeyboard("abcdefghijklmno",groups,"other",0,123,"AutoFilterMGbot",&config.Config{})
 if !strings.Contains(old[0][0].Text,"English Sub Files"){t.Fatalf("old other callback should redirect to English: %+v",old)}
}
