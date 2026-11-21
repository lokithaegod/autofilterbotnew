package mongo

import (
 "regexp"
 "testing"
)

func TestFilenameSearchPattern(t *testing.T) {
 query:="once upon a time in hollywood"
 names:=[]string{
  "Once.Upon.A.TimeIn.Hollywood.2019.720p.BluRay.x264-[YTS.LT].mp4",
  "Once.Upon.A.TimeIn.Hollywood.2019.1080p.BluRay.x264-[YTS.LT].mp4",
  "Once_Upon_a_Time_in_Hollywood_2019_720p_BluRay_H264_AAC_RARBG.mp4",
  "Once.Upon.a.Time.in.Hollywood.2019.1080p.BluRay.x265-RARBG.mp4",
 }
 pattern:=regexp.MustCompile(filenameSearchPattern(query))
 for _,name:=range names {
  if !pattern.MatchString(name) {t.Errorf("failed to match %q",name)}
 }
 if pattern.MatchString("Completely.Different.Movie.2019.mp4") {t.Fatal("matched unrelated file")}
}
func TestFilenameSearchEscapesRegex(t *testing.T) {
 pattern:=regexp.MustCompile(filenameSearchPattern("show (2024)"))
 if !pattern.MatchString("Show.(2024).mkv"){t.Fatal("expected literal parentheses match")}
 if regexp.MustCompile(filenameSearchPattern("")).MatchString("anything"){t.Fatal("empty query matched")}
}
