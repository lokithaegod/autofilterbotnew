package core

import "testing"

func TestForwardedMalayalamKind(t *testing.T) {
 cases:=[]struct{name string;nativeVideo bool;want string}{
  {"Cast.Away.2000.720p.BRRip.mkv",false,"video"},
  {"A.Quiet.Place.2018.1080p.mp4",false,"video"},
  {"Telegram_video",true,"video"},
  {"Cast.Away.2000.720p.srt",false,"srt"},
  {"Movie.2000.SRT",false,"srt"},
  {"Movie.2000.Malayalam.zip",false,""},
  {"Movie.2000.English.ass",false,""},
  {"archive.rar",false,""},
  {"",false,""},
 }
 for _,tc:=range cases{
  got:=ForwardedMalayalamKind(tc.name,tc.nativeVideo)
  if got!=tc.want{t.Errorf("%q native %v: got %q want %q",tc.name,tc.nativeVideo,got,tc.want)}
 }
}

func TestForwardedSRTDisplaysInMalayalamSection(t *testing.T){
 cases:=[]struct{name string;tagged,want bool}{
  {"Cast.Away.2000.720p.srt",true,true},
  {"Cast.Away.2000.720p.srt",false,false},
  {"Cast.Away.2000.720p.മലയാളം.srt",false,true},
  {"A.Quiet.Place.2018.Malayalam.zip",false,true},
  {"A.Quiet.Place.2018.English.zip",false,false},
  {"A.Quiet.Place.2018.1080p.mkv",true,true},
  {"A.Quiet.Place.2018.1080p.mkv",false,false},
 }
 for _,tc:=range cases{
  got:=isManuallySuitableMalayalamFile(tc.name,tc.tagged)
  if got!=tc.want{t.Errorf("%q tagged %v: got %v want %v",tc.name,tc.tagged,got,tc.want)}
 }
}

func TestAuthorizedCaptionChannel(t *testing.T){
 allowed:=[]int64{-1003931602584,-1001234567890}
 if !isAuthorizedCaptionChannel(-1003931602584,allowed){t.Fatal("known channel rejected")}
 if isAuthorizedCaptionChannel(-1009999999999,allowed){t.Fatal("unknown channel accepted")}
}
