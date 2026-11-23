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
