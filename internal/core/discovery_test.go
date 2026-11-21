package core

import "testing"

func TestDiscoveryFilters(t *testing.T) {
 got,err:=parseDiscover("Bruce Almighty year:2003 quality:1080p lang:eng sort:smallest minmb:300")
 if err!=nil {t.Fatal(err)}
 if got.Query!="Bruce Almighty" || got.Year!="2003" || got.Quality!="1080p" || got.Language!="eng" || got.Sort!="smallest" || got.MinSize!=300*1024*1024 {t.Fatalf("%+v",got)}
 if _,err:=parseDiscover("minmb:200 maxmb:100");err==nil {t.Fatal("size validation missing")}
 if _,err:=parseDiscover("year:xyz");err==nil {t.Fatal("year validation missing")}
}
func TestSuggestions(t *testing.T){
 if fuzzyDistance("bruce","bruse")!=1{t.Fatal("edit distance")}
 if normalizeTitle("Bruce.Almighty.2003.1080p.mkv")!="bruce almighty 1080p mkv"{t.Fatal("normalize")}
}
