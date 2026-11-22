package autofilter

import (
 "strings"
 "testing"

 "github.com/Jisin0/autofilterbot/internal/model"
)

func TestFringeEpisodeDisplayAndOrdering(t *testing.T){
 if got:=displayFileName("fri.s05e06.br720p.299mb-PaHe.in.mkv");got!="Fringe.s05e06.br720p.299mb-PaHe.in.mkv"{t.Fatal(got)}
 for _,name:=range []string{"Friday.S05E06.mkv","fri.movie.mkv","friends.s05e06.mkv"}{
  if displayFileName(name)!=name{t.Fatalf("unrelated title altered: %q",name)}
 }
 a:=parseEpisode("fri.s05e06.br720p.299mb-PaHe.in.mkv")
 b:=parseEpisode("Fringe.S04E13.PROPER.720p.BluRay.mkv")
 if a.title!="fringe"||b.title!="fringe"{t.Fatalf("episode title canonicalization: %+v %+v",a,b)}
 files:=[]Files{{{File:model.File{FileName:"fri.s05e06.mkv"}},{File:model.File{FileName:"Fringe.S04E13.mkv"}},{File:model.File{FileName:"fri.s01e01.mkv"}}}}
 SortEpisodes(files)
 got:=[]string{files[0][0].FileName,files[0][1].FileName,files[0][2].FileName}
 expected:=[]string{"fri.s01e01.mkv","Fringe.S04E13.mkv","fri.s05e06.mkv"}
 for i:=range got {if got[i]!=expected[i]{t.Fatalf("order %v expected %v",got,expected)}}
 // Original filenames remain unmodified, so Telegram's file IDs are stable.
 if strings.HasPrefix(files[0][2].FileName,"Fringe"){t.Fatal("original file modified")}
}
