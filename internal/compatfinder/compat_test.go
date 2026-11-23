package compatfinder

import "testing"

func TestIdentity(t *testing.T){
 cases:=[]struct{filename,title,year string}{
  {"Transformers.2.2009.IMAX.1080p.BRRip.mkv","transformers 2","2009"},
  {"War.Dogs.2016.720p.BluRay.x264-[YTS.AG].mp4","war dogs","2016"},
  {"Never.Surrender.2026.720p.WEB-DL.@Team_Goat_Bot.mkv","never surrender","2026"},
 }
 for _,c:=range cases{
  got,year,ok:=Identity(c.filename)
  if !ok||got!=c.title||year!=c.year{t.Errorf("%s: %s %s %v",c.filename,got,year,ok)}
 }
}
func TestNeverGuessVerification(t *testing.T){
 tests:=[]struct{subtitle,video,want string}{
  {"Movie.2005.Malayalam.srt","Movie.2005.1080p.BluRay.mkv","unverified"},
  {"Movie.2005.IMAX.Malayalam.srt","Movie.2005.Theatrical.720p.mkv","incompatible"},
  {"Movie.2005.srt","Movie.2006.mkv","incompatible"},
  {"Movie.Malayalam.srt","Movie.2005.mkv","unverified"},
  {"Transformers.2.2009.IMAX.srt","Transformers.2.2009.IMAX.720p.BluRay.mkv","unverified"},
 }
 for _,c:=range tests{
  got,_:=Classify(c.subtitle,c.video)
  if got!=c.want{t.Errorf("%s versus %s -> %s expected %s",c.subtitle,c.video,got,c.want)}
 }
}
