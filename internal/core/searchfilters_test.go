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

func TestMalayalamSubtitleFilter(t *testing.T) {
 cases:=[]struct{name string; subtitle,malayalam bool}{
  {"Bruce.Almighty.2003.BluRay.മലയാളം (1).srt",true,true},
  {"Prometheus.2012.1080p.BrRip.മലയാളം.srt",true,true},
  {"Film.MAL.SUB.srt",true,true},
  {"Film.MSub.ass",true,true},
  {"Film.Tamil.srt",true,false},
  {"Film.English.vtt",true,false},
  {"Film.720p.MAL.SUB.mkv",false,false},
  {"Film.720p.mp4",false,false},
 }
 for _,tt:=range cases {
  if got:=isSubtitleFile(tt.name);got!=tt.subtitle {t.Errorf("file %q subtitle %v want %v",tt.name,got,tt.subtitle)}
  if got:=isMalayalamSubtitle(tt.name);got!=tt.malayalam {t.Errorf("file %q malayalam %v want %v",tt.name,got,tt.malayalam)}
 }
 makeFile:=func(name string) autofilter.File{return autofilter.File{File:model.File{FileName:name}}}
 pages:=[]autofilter.Files{{makeFile("Bruce.Almighty.2003.720p.mp4"),makeFile("Bruce.Almighty.2003.BluRay.മലയാളം.srt"),makeFile("Bruce.Almighty.2003.Tamil.srt")}}
 if got:=matchingSubtitleFiles(pages,"all","all","all","msub");len(got)!=1||got[0].FileName!="Bruce.Almighty.2003.BluRay.മലയാളം.srt" {t.Errorf("malayalam results: %v",got)}
 if got:=matchingSubtitleFiles(pages,"all","all","all","sub");len(got)!=2 {t.Errorf("all subtitles: %v",got)}
 if got:=matchingSubtitleFiles(pages,"all","all","all","video");len(got)!=1||got[0].FileName!="Bruce.Almighty.2003.720p.mp4" {t.Errorf("videos: %v",got)}
 if got:=matchingSubtitleFiles(pages,"all","mal","all","msub");len(got)!=1 {t.Errorf("malayalam language and file type: %v",got)}
}
