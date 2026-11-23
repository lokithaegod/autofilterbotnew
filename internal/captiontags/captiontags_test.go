package captiontags
import "testing"

func TestIsTagged(t *testing.T){
 cases:=[]struct{caption string;want bool}{
  {"Malayalam Sub",true},
  {"✅ Malayalam Sub ✅",true},
  {"Malayalam subtitle available",true},
  {"MALAYALAM_SUB",true},
  {"Movie Name\nMalayalam Subs",true},
  {"English Sub",false},
  {"No Malayalam subtitles available",true}, // keywords alone are a declaration; avoid negating claims in captions.
  {"Malayalam Subscription",false},
  {"",false},
 }
 for _,tt:=range cases{if got:=IsTagged(tt.caption);got!=tt.want{t.Errorf("%q: got %v want %v",tt.caption,got,tt.want)}}
}
func TestVideoUpload(t *testing.T){
 for _,tt:=range []struct{name string;video,want bool}{
  {"movie.mkv",false,true},{"movie.mp4",false,true},{"movie.srt",false,false},
  {"movie.ass",false,false},{"other.zip",false,false},
  {"Movie",true,true},
 }{
  if got:=IsVideoUpload(tt.name,tt.video);got!=tt.want{t.Errorf("%+v got %v",tt,got)}
 }
}
