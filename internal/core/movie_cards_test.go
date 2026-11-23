package core

import (
 "strings"
 "testing"

 "github.com/Jisin0/autofilterbot/internal/autofilter"
 "github.com/Jisin0/autofilterbot/internal/model"
 "github.com/Jisin0/autofilterbot/internal/config"
)

func mockFile(name string)autofilter.File{return autofilter.File{File:model.File{FileName:name}}}

func TestMovieCardsGroupingSubtitlesAndYears(t *testing.T){
 pages:=[]autofilter.Files{
  {mockFile("Bruce.Almighty.2003.720p.BrRip.x264.YIFY.mp4"),
   mockFile("Bruce.Almighty.2003.1080p.BluRay.mkv"),
   mockFile("Bruce.Almighty.2003.BluRay.മലയാളം (1).srt")},
  {mockFile("Bruce.Almighty.2003.Tamil.srt"),mockFile("Bruce.Almighty.2010.720p.mkv"),
   mockFile("Prometheus.2012.720p.mkv")},
 }
 cards:=buildMovieCards(pages)
 if len(cards)!=3{t.Fatalf("expected 3 distinct movie year groups, got %d: %#v",len(cards),cards)}
 c:=cards[0]
 if c.Title!="Bruce Almighty"||c.Year!="2003"{t.Errorf("incorrect movie title: %q (%s)",c.Title,c.Year)}
 if len(c.Videos)!=2||len(c.Subs)!=2||c.MalayalamSubs!=1{t.Errorf("incorrect subtype groups: %+v",c)}
 if strings.Contains(cardLabel(c),"✅📜"){t.Errorf("filename-based subtitle badge must be disabled: %s",cardLabel(c))}
 if cards[1].Year!="2010"||cards[1].MalayalamSubs!=0{t.Errorf("subtitle badge leaked between years: %+v",cards[1])}
 if strings.Contains(cardLabel(cards[2]),"✅📜"){t.Errorf("other films must not inherit subtitle badge")}
}

func TestMovieCardsUnmatchedNamesAndEpisodes(t *testing.T){
 pages:=[]autofilter.Files{{mockFile("Mystery.Movie.720p.mkv"),mockFile("Mystery.Movie.1080p.mkv")}}
 if got:=len(buildMovieCards(pages));got!=2{t.Fatalf("unmatched filenames must not merge, got %d",got)}
 if hasEpisodes(pages){t.Fatal("movies incorrectly recognized as episodes")}
 episodes:=[]autofilter.Files{{mockFile("Ted.Lasso.S01E01.720p.mkv"),mockFile("Ted.Lasso.S01E02.1080p.mkv")}}
 if !hasEpisodes(episodes){t.Fatal("episodes not recognized")}
 if got:=len(buildMovieCards(episodes));got!=2{t.Fatalf("episodes must not be grouped as movies, got %d",got)}
}

func TestMovieCardKeyboardAndPages(t *testing.T){
 cards:=buildMovieCards([]autofilter.Files{{mockFile("Bruce.Almighty.2003.720p.mkv"),mockFile("Bruce.Almighty.2003.മലയാളം.srt")}})
 keyboard:=movieCardsKeyboard("abcdefghijklmno",cards,0)
 if len(keyboard)<3 {t.Fatalf("missing navigation or switch to file view: %+v",keyboard)}
 if strings.Contains(keyboard[1][0].Text,"✅📜"){t.Fatalf("caption-only cards must not show filename-based badge: %q",keyboard[1][0].Text)}
 if keyboard[0][0].CallbackData!="navg|abcdefghijklmno_0"{t.Fatal("original file list link changed")}
 if len(keyboard[1][0].CallbackData)>64{t.Fatal("callback data over Telegram's 64 byte limit")}
 detail:=movieDetailKeyboard("abcdefghijklmno",0,0,cards[0],123,"AutofilterMGbot","all",&config.Config{})
 if len(detail)<5 {t.Fatalf("caption-only cards must retain standard movie navigation: %+v",detail)}
 if !strings.Contains(detail[1][0].Text,"Malayalam Sub Files") {t.Fatalf("caption-only button missing: %+v",detail)}
 if !strings.Contains(detail[2][0].Text,"More Files") {t.Fatalf("more files button missing: %+v",detail)}
 if len(detail)!=5 {t.Fatalf("movie card should show category choices before file links: %+v",detail)}
}

func TestSameMovieWithDifferentSourceChannelLabels(t *testing.T) {
 pages:=[]autofilter.Files{
  {mockFile("Inception.2010.1080p.BluRay.mkv"),mockFile("@CC Links Inception.2010.720p.BRRip.mp4")},
  {mockFile("[CC].Inception.2010.720p.BluRay.mkv"),mockFile("Inception.2011.1080p.mkv")},
 }
 cards:=buildMovieCards(pages)
 if len(cards)!=2{t.Fatalf("expected a single 2010 card, plus separate 2011 movie: %+v",cards)}
 if cards[0].Title!="Inception"||cards[0].Year!="2010"{t.Fatalf("wrong canonical movie: %+v",cards[0])}
 if total:=len(cards[0].Videos)+len(cards[0].Other)+len(cards[0].Subs);total!=3{t.Fatalf("merged card should contain 3 releases, got %d",total)}
 if cards[1].Year!="2011"{t.Fatalf("different release year incorrectly merged: %+v",cards)}
 for _,name:=range []string{"@CC Links Inception.2010.1080p.mkv","[CC].Inception.2010.720p.mkv"}{
  title,year,key,ok:=titleFromFilename(name)
  if !ok||title!="Inception"||year!="2010"||key!="inception|2010"{t.Errorf("bad title cleanup for %s: %q, %q, %q, %v",name,title,year,key,ok)}
 }
}

func TestMovieCategoryChoiceButtons(t *testing.T) {
 cards:=buildMovieCards([]autofilter.Files{{mockFile("Inception.2010.1080p.mkv")}})
 menu:=movieDetailKeyboard("abcdefghijklmno",0,0,cards[0],123,"AutofilterMGbot","all",&config.Config{})
 if len(menu)!=5 {t.Fatalf("only two category buttons and navigation should be visible: %+v",menu)}
 if menu[1][0].CallbackData!="card|abcdefghijklmno_0_0_tagged" {t.Fatalf("tagged callback changed: %+v",menu[1])}
 if menu[2][0].CallbackData!="card|abcdefghijklmno_0_0_other" {t.Fatalf("other callback changed: %+v",menu[2])}
 other:=movieDetailKeyboard("abcdefghijklmno",0,0,cards[0],123,"AutofilterMGbot","other",&config.Config{})
 hasBack:=false
 for _,row:=range other{for _,button:=range row{if button.Text=="↩️ File categories"{hasBack=true}}}
 if !hasBack{t.Fatalf("missing return to file categories: %+v",other)}
}
