package captiontags

import "testing"

func TestTamilDubbedCaptionRecognition(t *testing.T){
 cases:=[]struct{caption string;want bool}{
  {"Tamil Dubbed",true},
  {"Tamil Dub",true},
  {"TAMIL DUBBING | 1080p",true},
  {"HD • Tamil-Dubbed • BluRay",true},
  {"தமிழ் டப்",true},
  {"Tamil Subtitles",false},
  {"Tamil Audio",true},
  {"Tamil ",true},
  {"MP3 2.0 — 116 kbps — Tamil",true},
  {"No Tamil Audio",false},
  {"Tamil Sub",false},
  {"No Tamil Dubbed",false},
  {"without Tamil Dub",false},
  {"Tamil Dub not available",false},
  {"Malayalam Sub",false},
  {"English Sub",false},
  {"",false},
 }
 for _,tc:=range cases{
  if got:=IsTamilDubbedCaption(tc.caption);got!=tc.want{
   t.Errorf("%q: got %v, want %v",tc.caption,got,tc.want)
  }
 }
}
