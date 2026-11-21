package autofilter

import (
 "testing"
 "github.com/Jisin0/autofilterbot/internal/model"
)

func TestEpisodeParsing(t *testing.T) {
 cases:=[]struct{name,title string;season,episode int;valid bool}{
  {"Ted.Lasso.S01E02.Biscuits.1080p.mkv","ted lasso",1,2,true},
  {"Ted.Lasso.S01E10.720p.mkv","ted lasso",1,10,true},
  {"The.100.S07E16.720p.mkv","the 100",7,16,true},
  {"Show.2x03.English.mp4","show",2,3,true},
  {"Some.Movie.2003.1080p.mkv","",0,0,false},
 }
 for _,tc:=range cases{
  got:=parseEpisode(tc.name)
  if got.valid!=tc.valid|| (tc.valid&&(got.title!=tc.title||got.season!=tc.season||got.episode!=tc.episode)){
   t.Errorf("parseEpisode(%q) = %#v",tc.name,got)
  }
 }
}
func TestEpisodeNaturalOrderAcrossPages(t *testing.T){
 makeFile:=func(name string) File{return File{File:model.File{FileName:name}}}
 pages:=[]Files{
  {makeFile("Ted.Lasso.S01E10.mkv"),makeFile("Ted.Lasso.S02E01.mkv")},
  {makeFile("Ted.Lasso.S01E02.mkv"),makeFile("Ted.Lasso.S01E01.mkv")},
 }
 SortEpisodes(pages)
 expected:=[]string{"Ted.Lasso.S01E01.mkv","Ted.Lasso.S01E02.mkv","Ted.Lasso.S01E10.mkv","Ted.Lasso.S02E01.mkv"}
 k:=0
 for _,page:=range pages{for _,file:=range page{
  if file.FileName!=expected[k]{t.Errorf("at %d got %q want %q",k,file.FileName,expected[k])}
  k++
 }}
}
func TestNonEpisodeOrderRemains(t *testing.T){
 pages:=[]Files{{{File:model.File{FileName:"Film B"}},{File:model.File{FileName:"Film A"}}}}
 SortEpisodes(pages)
 if pages[0][0].FileName!="Film B" {t.Fatal("non-episode results reordered")}
}
