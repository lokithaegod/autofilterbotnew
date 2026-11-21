package core

import (
 "testing"
 "github.com/Jisin0/autofilterbot/internal/autofilter"
 "github.com/Jisin0/autofilterbot/internal/model"
)
func TestQualityAndLanguage(t *testing.T) {
 cases:=[]struct{name,q,l string;want bool}{
  {"Test.Movie.1080p.Malayalam.mkv","1080p","mal",true},
  {"Movie.720p.Tamil.mp4","1080p","tam",false},
  {"Movie.4K.English.mkv","2160p","eng",true},
  {"Movie.2160p.Hindi.mkv","2160p","hin",true},
  {"Movie.720p.Multi.mkv","720p","mal",false},
  {"Movie.1080p.MAL.mkv","1080p","mal",true},
 }
 for _,tt:=range cases {
  got:=qualityMatches(tt.name,tt.q)&&languageMatches(tt.name,tt.l)
  if got!=tt.want {t.Errorf("%s q=%s l=%s got %v want %v",tt.name,tt.q,tt.l,got,tt.want)}
 }
 pages:=[]autofilter.Files{{{File:model.File{FileName:"A.720p.Tamil.mkv"}},{File:model.File{FileName:"B.1080p.English.mkv"}}}}
 if got:=matchingFiles(pages,"720p","tam");len(got)!=1||got[0].FileName!="A.720p.Tamil.mkv"{t.Fatalf("unexpected filter result: %#v",got)}
}
