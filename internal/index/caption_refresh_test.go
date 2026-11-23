package index

import (
 "testing"
 "github.com/amarnathcjd/gogram/telegram"
)

func TestScanCaptionMessagesOnlyMarksCaptionedVideos(t *testing.T){
 cases:=[]struct{name,caption,filename string;want int}{
  {"movie caption","Malayalam Sub","A.Quiet.Place.2018.1080p.mkv",1},
  {"another form","✅ Malayalam Subs","Cast.Away.2000.720p.mp4",1},
  {"no caption","English subtitles","Movie.2000.1080p.mkv",0},
  {"negative caption","No Malayalam subtitles available","Movie.2000.mkv",0},
  {"subtitle isn't video","Malayalam Sub","Movie.2000.srt",0},
  {"zip isn't video","Malayalam Sub","Movie.2000.മലയാളം.zip",0},
 }
 for _,tc:=range cases{
  t.Run(tc.name,func(t *testing.T){
   history:=&telegram.MessagesMessagesObj{Messages:[]telegram.Message{
    &telegram.MessageObj{ID:17,Message:tc.caption,Media:&telegram.MessageMediaDocument{
     Document:&telegram.DocumentObj{Size:100,Attributes:[]telegram.DocumentAttribute{
      &telegram.DocumentAttributeFilename{FileName:tc.filename},
     }},
    }},
   }}
   found:=0
   result:=CaptionScanResult{}
   err:=scanCaptionMessages(history,-1001234567890,func(id,name string,size int64,enabled bool)(int,error){
    found++
    if id==""||name!=tc.filename||size!=100||!enabled{t.Errorf("unexpected tag data: %q %q %d %v",id,name,size,enabled)}
    return 2,nil
   },&result)
   if err!=nil{t.Fatal(err)}
   if found!=tc.want||result.Captioned!=tc.want||result.Scanned!=1{
    t.Fatalf("found %d scanned %d captioned %d, want %d",found,result.Scanned,result.Captioned,tc.want)
   }
  })
 }
}
