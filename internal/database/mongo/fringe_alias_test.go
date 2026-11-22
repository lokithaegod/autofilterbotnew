package mongo

import (
 "regexp"
 "testing"

 "go.mongodb.org/mongo-driver/bson"
)

func aliasRegex(t *testing.T,query string) string {
 t.Helper()
 f:=fileSearchFilter(query)
 if len(f)!=1||f[0].Key!="$or"{t.Fatalf("%q should include original and Fringe alias: %#v",query,f)}
 branches,ok:=f[0].Value.(bson.A)
 if !ok||len(branches)!=2{t.Fatalf("bad regex branches: %#v",f)}
 alias:=branches[1].(bson.D)[0].Value.(bson.D)[0].Value.(string)
 return alias
}

func TestFringeAliasIncludesOnlyShortSeriesEpisodes(t *testing.T) {
 rx:=regexp.MustCompile(aliasRegex(t,"Fringe"))
 for _,name:=range []string{
  "fri.s01e01.br720p.550mb-PaHe.in.mkv",
  "fri.s05e06.br720p.299mb-PaHe.in.mkv",
  "fri.S05E09.br720p.mkv",
 }{
  if !rx.MatchString(name){t.Errorf("%q should match Fringe alias",name)}
 }
 for _,name:=range []string{
  "Friday.2020.mp4",
  "friends.s05e06.mkv",
  "fri.movie.2020.mkv",
  "friday.s05e06.mp4",
  "other.fri.s05e06.mkv",
 }{
  if rx.MatchString(name){t.Errorf("%q must not match Fringe alias",name)}
 }
}

func TestFringeEpisodeTargetedAlias(t *testing.T) {
 rx:=regexp.MustCompile(aliasRegex(t,"fringe S05E06"))
 if !rx.MatchString("fri.s05e06.br720p.299mb-PaHe.in.mkv"){t.Fatal("targeted episode missing")}
 if rx.MatchString("fri.s05e07.br720p.299mb-PaHe.in.mkv"){t.Fatal("other episode matched targeted search")}
 for _,query:=range []string{"fringe movie","Friday","fRi","Fringe S01","Fringe 2020"}{
  f:=fileSearchFilter(query)
  if f[0].Key!="_file_name"&&f[0].Key!="file_name"{t.Errorf("%q must not trigger alias: %#v",query,f)}
 }
}
