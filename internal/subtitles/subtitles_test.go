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

 
func TestRealisticScreenshotReleaseNames(t *testing.T) {
    cases := []struct {
        name string
        wantType string
        wantGroup string
        wantResolution string
    }{
        {"War.Dogs.2016.720p.BluRay.x264-[YTS.AG].mp4", "BluRay", "yts", "720p"},
        {"Never.Surrender.2026.720p.WEB-DL.@Team_Goat_Bot.mkv", "WEB-DL", "", "720p"},
        {"Transformers.2.2009.IMAX.1080p.BrRip.mkv", "BRRip", "", "1080p"},
        {"Transformers.2.2009.IMAX.HQ.720p.BluRay.mkv", "BluRay", "", "720p"},
        {"Movie.2020.1080p.BDRip.mkv", "BDRip", "", "1080p"},
        {"Movie.2020.1080p.HDRip.mkv", "HDRip", "", "1080p"},
        {"Movie.2020.1080p.BRRip.YTS.srt", "BRRip", "yts", "1080p"},
    }
    for _, tt := range cases {
        got := ReleaseFromFilename(tt.name)
        if got.Type != tt.wantType || got.Group != tt.wantGroup || got.Resolution != tt.wantResolution {
            t.Errorf("%s: got %+v; expected type=%s group=%s resolution=%s", tt.name, got, tt.wantType, tt.wantGroup, tt.wantResolution)
        }
    }
}
func TestDiscRelatedButNotProvenSynced(t *testing.T) {
    cases := []struct {
        video, subtitle, level string
    }{
        {"Transformers.2.2009.IMAX.1080p.BRRip.mkv", "Transformers.2.2009.IMAX.1080p.BluRay.srt", "possible"},
        {"Transformers.2.2009.IMAX.720p.BluRay.mkv", "Transformers.2.2009.IMAX.1080p.BRRip.srt", "possible"},
        {"Movie.2020.BDRip.mkv", "Movie.2020.BRRip.srt", "possible"},
        {"Movie.2020.BluRay.mkv", "Movie.2020.HDRip.srt", "different"},
        {"Movie.2020.BluRay.mkv", "Movie.2020.WEB-DL.srt", "different"},
        {"Movie.2020.BRRip.25fps.mkv", "Movie.2020.BluRay.23.976fps.srt", "different"},
        {"Never.Surrender.2026.720p.WEB-DL.@Team_Goat_Bot.mkv", "Never.Surrender.2026.720p.WEB-DL.srt", "likely"},
        {"War.Dogs.2016.720p.BluRay.x264-[YTS.AG].mp4", "War.Dogs.2016.720p.BRRip.YTS.srt", "possible"},
    }
    for _, tt := range cases {
        level, _, _ := Compare(ReleaseFromFilename(tt.video), ReleaseFromFilename(tt.subtitle))
        if level != tt.level {
            t.Errorf("video %s with subtitle %s: got %s, want %s", tt.video, tt.subtitle, level, tt.level)
        }
    }
}
func TestTransformersTitleAndYear(t *testing.T) {
    title, year, key, ok := Parse("Transformers.2.2009.IMAX.1080p.BRRip.Malayalam.srt")
    if !ok || title != "Transformers 2" || year != "2009" || key != "transformers 2" {
        t.Errorf("unexpected parsed identity: %q %q %q %v", title, year, key, ok)
    }
}
