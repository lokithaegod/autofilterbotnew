package core

import (
 "context"
 "encoding/json"
 "fmt"
 "io"
 "net/http"
 "os"
 "os/exec"
 "path/filepath"
 "strconv"
 "strings"
 "time"

 "github.com/Jisin0/autofilterbot/internal/compatfinder"
 "github.com/Jisin0/autofilterbot/internal/fsub"
 "github.com/Jisin0/autofilterbot/internal/model"
 "github.com/Jisin0/autofilterbot/internal/subtitles"
 "github.com/PaulSonOfLars/gotgbot/v2"
 "github.com/PaulSonOfLars/gotgbot/v2/ext"
)

// Existing MongoDB file records are read-only. The optional manifest maps
// existing Telegram file_unique_id values to accessible local video files.
// No full movie is ever fetched from Telegram by this feature.
func compatibleMediaManifest()(map[string]string,error){
 path:=os.Getenv("COMPAT_MEDIA_MANIFEST")
 if path==""{return map[string]string{},nil}
 data,err:=os.ReadFile(path)
 if err!=nil{return nil,err}
 var files map[string]string
 if err=json.Unmarshal(data,&files);err!=nil{return nil,err}
 return files,nil
}

func downloadSelectedSRT(bot *gotgbot.Bot,fileID string)(string,error){
 file,err:=bot.GetFile(fileID,nil)
 if err!=nil{return "",err}
 if file.FileSize>4*1024*1024{return "",fmt.Errorf("SRT exceeds 4MB safety limit")}
 ctx,cancel:=context.WithTimeout(context.Background(),30*time.Second)
 defer cancel()
 req,err:=http.NewRequestWithContext(ctx,"GET",file.URL(bot,nil),nil)
 if err!=nil{return "",err}
 resp,err:=http.DefaultClient.Do(req)
 if err!=nil{return "",fmt.Errorf("subtitle download failed")}
 defer resp.Body.Close()
 if resp.StatusCode!=200{return "",fmt.Errorf("subtitle download unavailable (HTTP %d)",resp.StatusCode)}
 tmp,err:=os.CreateTemp("","compat-selected-*.srt")
 if err!=nil{return "",err}
 defer tmp.Close()
 if err=tmp.Chmod(0600);err!=nil{os.Remove(tmp.Name());return "",err}
 n,err:=io.Copy(tmp,io.LimitReader(resp.Body,4*1024*1024+1))
 if err!=nil||n>4*1024*1024{os.Remove(tmp.Name());return "",fmt.Errorf("subtitle exceeded the allowed size")}
 return tmp.Name(),nil
}

type compatProbeResult struct {
 Status string `json:"status"`
 Reason string `json:"reason"`
 Confidence string `json:"confidence"`
 Offset float64 `json:"offset_seconds"`
 Drift float64 `json:"drift_seconds"`
}
func runCompatibilityProbe(subtitlePath,videoPath string)compatProbeResult{
 if videoPath==""||!filepath.IsAbs(videoPath){return compatProbeResult{Status:"unverified",Reason:"No authorized local media path for audio analysis"}}
 if info,err:=os.Stat(videoPath);err!=nil||!info.Mode().IsRegular(){
  return compatProbeResult{Status:"unverified",Reason:"Video is not accessible on the analysis host"}
 }
 ctx,cancel:=context.WithTimeout(context.Background(),4*time.Minute)
 defer cancel()
 out,err:=exec.CommandContext(ctx,"python3","scripts/compatibility_probe.py",
  "--subtitle",subtitlePath,"--video",videoPath).Output()
 if err!=nil{return compatProbeResult{Status:"unverified",Reason:"FFprobe/FFsubsync unavailable or analysis failed"}}
 var result compatProbeResult
 if json.Unmarshal(out,&result)!=nil{return compatProbeResult{Status:"unverified",Reason:"Invalid audio analysis output"}}
 switch result.Status{case "verified","needs_sync","incompatible","unverified":
 default:return compatProbeResult{Status:"unverified",Reason:"Unknown analysis state"}}
 return result
}

func findCompatibleVideos(bot *gotgbot.Bot,chatID,userID int64,filename,fileID string)error{
 if !subtitles.Supported(filename)||!strings.EqualFold(filepath.Ext(filename),".srt"){
  _,err:=bot.SendMessage(chatID,"Please select a .srt subtitle file.",nil);return err
 }
 videos,err:=_app.DB.FindCompatibilityCandidates(filename)
 if err!=nil{
  _,sendErr:=bot.SendMessage(chatID,"Cannot search every movie record: "+err.Error(),nil)
  return sendErr
 }
 paths,manifestErr:=compatibleMediaManifest()
 if manifestErr!=nil{paths=map[string]string{}}
 selected,err:=downloadSelectedSRT(bot,fileID)
 if err!=nil{
  // No fake 'verified' results when subtitle bytes cannot be read.
  selected=""
 }
 if selected!=""{defer os.Remove(selected)}
 groups:=map[string][]string{"verified":{},"needs_sync":{},"incompatible":{},"unverified":{}}
 const maxAudioChecks=12 // Bound expensive audio work in each user request.
 analyzed:=0
 for _,video:=range videos{
  status,reason:=compatfinder.Classify(filename,video.FileName)
  if status=="unverified"&&video.FileType=="document"{
   reason="Legacy document file extension is unavailable; type not confirmed"
  }
  var probe compatProbeResult
  if status=="unverified"&&selected!=""&&paths[video.UniqueId]!=""&&analyzed<maxAudioChecks{
   analyzed++
   probe=runCompatibilityProbe(selected,paths[video.UniqueId])
   if probe.Status!=""{status,reason=probe.Status,probe.Reason}
  }
  // Only actual media-analysis evidence can mark a file verified.
  if status=="verified" && probe.Status!="verified"{status="unverified"}
  display:=video.FileName
  if len([]rune(display))>85{display=string([]rune(display)[:85])+"…"}
  line:="• "+display+" — "+reason
  if probe.Status=="verified"||probe.Status=="needs_sync"{
   line+=fmt.Sprintf(" (offset %.2fs, drift %.2fs)",probe.Offset,probe.Drift)
  }
  groups[status]=append(groups[status],line)
 }
 sections:=[]struct{key,title string}{
  {"verified","✅ VERIFIED TIMING MATCHES"},
  {"needs_sync","⚠️ NEEDS SYNCHRONIZATION"},
  {"incompatible","❌ INCOMPATIBLE"},
  {"unverified","❔ OTHER FILES — NOT YET VERIFIED"},
 }
 title,year,_:=compatfinder.Identity(filename)
 header:=fmt.Sprintf("🎬 MOVIE: %s (%s)\n📄 SELECTED SRT: %s\nCandidate records: %d\nAudio analyzed: %d\n",
  title,year,filename,len(videos),analyzed)
 if selected==""{header+="\nSubtitle content unavailable to analyzer: verification was not run.\n"}
 if manifestErr!=nil{header+="\nMedia manifest unavailable: verification disabled.\n"}
 _,err=bot.SendMessage(chatID,header,nil)
 if err!=nil{return err}
 for _,section:=range sections{
  entries:=groups[section.key]
  if len(entries)==0{continue}
  // Telegram text is limited; use bounded chunks, never claim the list is complete when capped.
  for offset:=0;offset<len(entries);{
   text:=section.title+"\n"
   next:=offset
   for next<len(entries)&&len(text)+len(entries[next])+2<=3500&&next-offset<25{
    text+=entries[next]+"\n"
    next++
   }
   if next==offset{next++;text+="• Filename too long to display\n"}
   if _,err=bot.SendMessage(chatID,text,nil);err!=nil{return err}
   offset=next
  }
 }
 if len(videos)==0{_,err=bot.SendMessage(chatID,"No matching indexed movie files found.",nil)}
 return err
}

// Run on /findcompatible when replying to an SRT document.
func FindCompatibleCommand(bot *gotgbot.Bot,ctx *ext.Context)error{
 m:=ctx.Message
 if m==nil||m.From==nil{return nil}
 blocked,err:=_app.DB.IsUserBlocked(m.From.Id)
 if err!=nil||blocked{return err}
 if m.ReplyToMessage==nil||m.ReplyToMessage.Document==nil{
  _,err=bot.SendMessage(m.Chat.Id,"Reply to a Malayalam .srt file with /findcompatible, or use Find Compatible Videos in the subtitles section.",nil)
  return err
 }
 doc:=m.ReplyToMessage.Document
 return findCompatibleVideos(bot,m.Chat.Id,m.From.Id,doc.FileName,doc.FileId)
}

// A selected existing subtitle record reuses its Telegram file ID, not a new record.
func FindCompatibleSubtitleCallback(bot *gotgbot.Bot,ctx *ext.Context)error{
 c:=ctx.CallbackQuery
 if c==nil{return nil}
 raw:=strings.TrimPrefix(c.Data,"cfinder|")
 pos:=strings.LastIndex(raw,"_")
 if pos<=0{return nil}
 owner,err:=strconv.ParseInt(raw[pos+1:],10,64)
 if err!=nil||owner!=c.From.Id{
  _,_=c.Answer(bot,&gotgbot.AnswerCallbackQueryOpts{Text:"Only the requesting user can use this button.",ShowAlert:true})
  return nil
 }
 blocked,err:=_app.DB.IsUserBlocked(c.From.Id)
 if err!=nil||blocked{return err}
 entry,err:=_app.DB.GetSubtitle(raw[:pos])
 if err!=nil{return err}
 _,_=c.Answer(bot,nil)
 return findCompatibleVideos(bot,c.Message.GetChat().Id,c.From.Id,entry.FileName,entry.FileID)
}

// A directly sent private SRT can start the finder without saving it to Mongo.
func FindCompatibleIncomingSRT(bot *gotgbot.Bot,ctx *ext.Context)error{
 m:=ctx.Message
 if m==nil||m.From==nil||m.Document==nil||!strings.EqualFold(filepath.Ext(m.Document.FileName),".srt"){return nil}
 if m.Chat.Type!="private"{return nil}
 blocked,err:=_app.DB.IsUserBlocked(m.From.Id)
 if err!=nil||blocked{return err}
 return findCompatibleVideos(bot,m.Chat.Id,m.From.Id,m.Document.FileName,m.Document.FileId)
}

// Explicitly reference existing media model to preserve the database's
// established file identity semantics.
var _ *model.File
var _ = fsub.CheckFsub
