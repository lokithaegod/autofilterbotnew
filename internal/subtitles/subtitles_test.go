package subtitles

import "testing"

func TestParse(t *testing.T){
 tests:=[]struct{name,title,year,key string;ok bool}{
  {"The.Descent.2005.Malayalam.srt","The Descent","2005","the descent",true},
  {"The_Descent_(2005)_Malayalam.ass","The Descent","2005","the descent",true},
  {"The Descent 2005 ml.vtt","The Descent","2005","the descent",true},
  {"The Descent 2005.mp4","","","",false},
  {"Unknown Movie Malayalam.srt","","","",false},
 }
 for _,tt:=range tests{
  title,year,key,ok:=Parse(tt.name)
  if title!=tt.title||year!=tt.year||key!=tt.key||ok!=tt.ok{
   t.Errorf("%q got %q %q %q %v",tt.name,title,year,key,ok)
  }
 }
}
func TestSearchTerms(t *testing.T){
 for _,tt:=range []struct{query,key,year string}{
  {"The Descent (2005)","the descent","2005"},
  {"The.Descent.2005","the descent","2005"},
  {"The Descent","the descent",""},
 }{
  k,y:=SearchTerms(tt.query)
  if k!=tt.key||y!=tt.year{t.Errorf("%q: %q %q",tt.query,k,y)}
 }
}

func TestReleaseFromFilename(t *testing.T){
 tests:=[]struct{file,kind,res,group,fps string}{
 {"Movie.2005.1080p.BluRay-ABC.mkv","BluRay","1080p","abc",""},
 {"Movie.2005.WEB-DL.720p-XYZ.srt","WEB-DL","720p","xyz",""},
 {"Movie.2005.WEBRip.25fps.srt","WEBRip","","","25"},
 {"Movie 2005 Malayalam.srt","","","",""},
 }
 for _,tt:=range tests{r:=ReleaseFromFilename(tt.file);if r.Type!=tt.kind||r.Resolution!=tt.res||r.Group!=tt.group||r.FPS!=tt.fps{t.Errorf("%s: %+v",tt.file,r)}}
}
func TestCompare(t *testing.T){
 video:=ReleaseFromFilename("Movie.2005.1080p.BluRay-ABC.mkv")
 exact:=ReleaseFromFilename("Movie.2005.1080p.BluRay-ABC.srt")
 diff:=ReleaseFromFilename("Movie.2005.WEB-DL.1080p-XYZ.srt")
 unknown:=ReleaseFromFilename("Movie.2005.Malayalam.srt")
 if level,_,_:=Compare(video,exact);level!="likely"{t.Errorf("same source got %s",level)}
 if level,_,_:=Compare(video,diff);level!="different"{t.Errorf("different source got %s",level)}
 if level,_,_:=Compare(video,unknown);level!="unknown"{t.Errorf("unknown source got %s",level)}
}
func TestRank(t *testing.T){
 mk:=func(s string)*Entry{return &Entry{FileName:s}}
 options:=[]*Entry{mk("Movie.2005.WEB-DL.srt"),mk("Movie.2005.BluRay.srt"),mk("Movie.2005.Malayalam.srt")}
 ranked:=Rank(options,[]string{"Movie.2005.1080p.BluRay.mkv"})
 if ranked[0].Entry!=options[1]||ranked[0].Level!="likely"{t.Errorf("ranking failed: %#v",ranked)}
 if ranked[2].Level!="different"{t.Errorf("mismatch not flagged: %#v",ranked)}
}
