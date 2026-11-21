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

func TestSeasonFilterAndOrder(t *testing.T) {
 makeFile:=func(name string) autofilter.File{return autofilter.File{File:model.File{FileName:name}}}
 pages:=[]autofilter.Files{
  {makeFile("Ted.Lasso.S04E02.1080p.mkv"),makeFile("Ted.Lasso.S01E10.720p.mkv")},
  {makeFile("Ted.Lasso.S01E02.1080p.mkv"),makeFile("Ted.Lasso.S02E01.720p.mkv"),makeFile("Ted.Lasso.S01E01.720p.mkv")},
 }
 autofilter.SortEpisodes(pages)
 seasons:=availableSeasons(pages)
 if len(seasons)!=3||seasons[0]!=1||seasons[1]!=2||seasons[2]!=4 {t.Fatalf("seasons: %v",seasons)}
 all:=matchingSeasonFiles(pages,"all","all","all")
 if len(all)!=5||all[0].FileName!="Ted.Lasso.S01E01.720p.mkv"||all[1].FileName!="Ted.Lasso.S01E02.1080p.mkv"{t.Fatalf("unexpected order: %+v",all)}
 seasonOne:=matchingSeasonFiles(pages,"all","all","1")
 if len(seasonOne)!=3 {t.Fatalf("expected 3 season one versions; got %d",len(seasonOne))}
 seasonOneHD:=matchingSeasonFiles(pages,"1080p","all","1")
 if len(seasonOneHD)!=1||seasonOneHD[0].FileName!="Ted.Lasso.S01E02.1080p.mkv"{t.Fatalf("bad combined filter: %+v",seasonOneHD)}
 noSeason:=matchingSeasonFiles(pages,"all","all","3")
 if len(noSeason)!=0{t.Fatalf("missing season should be empty")}
}
