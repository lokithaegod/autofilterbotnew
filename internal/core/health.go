package core

import (
 "context"
 "fmt"
 "net/http"
 "time"

 "github.com/PaulSonOfLars/gotgbot/v2"
 "github.com/PaulSonOfLars/gotgbot/v2/ext"
)

// A positive status means polling was started AND MongoDB and Telegram's
// Bot API are reachable. A bot that is blocked or whose polling goroutine
// has stalled without producing an error cannot be detected conclusively.
type HealthStatus struct {
 Started bool
 Mongo bool
 Telegram bool
}
func (s HealthStatus) Healthy()bool{return s.Started&&s.Mongo&&s.Telegram}

type probeResult struct {name string;err error}
func checkHealth(ctx context.Context,started bool,dbCheck func(context.Context)error,telegramCheck func()error) HealthStatus {
 status:=HealthStatus{Started:started}
 if !started {return status}
 outcomes:=make(chan probeResult,2)
 go func(){outcomes<-probeResult{name:"mongo",err:dbCheck(ctx)}}()
 go func(){outcomes<-probeResult{name:"telegram",err:telegramCheck()}}()
 for i:=0;i<2;i++ {
  select{
  case r:=<-outcomes:
   if r.name=="mongo"{status.Mongo=r.err==nil}else{status.Telegram=r.err==nil}
  case <-ctx.Done():return status
  }
 }
 return status
}

func currentHealth(ctx context.Context)HealthStatus{
 app:=_app
 if app==nil||app.DB==nil||app.Bot==nil{return HealthStatus{}}
 return checkHealth(ctx,app.pollingStarted.Load(),app.DB.Ping,func()error{
  _,err:=app.Bot.GetMe(nil)
  return err
 })
}

// HealthHTTP serves both "/" (already used by UptimeRobot and GitHub Actions)
// and "/healthz". The legacy success text stays identical for compatibility.
// It returns 503 for a missing bot, failed database check or Bot API failure.
// This endpoint intentionally exposes no credentials or admin statistics.
func HealthHTTP(w http.ResponseWriter,r *http.Request){
 if r.Method!=http.MethodGet&&r.Method!=http.MethodHead{
  w.WriteHeader(http.StatusMethodNotAllowed);return
 }
 ctx,cancel:=context.WithTimeout(r.Context(),5*time.Second)
 defer cancel()
 status:=currentHealth(ctx)
 w.Header().Set("Content-Type","text/plain; charset=utf-8")
 w.Header().Set("Cache-Control","no-store")
 if !status.Healthy(){
  w.WriteHeader(http.StatusServiceUnavailable)
  _,_=w.Write([]byte("Telegram bot or database not healthy"))
  return
 }
 w.WriteHeader(http.StatusOK)
 _,_=w.Write([]byte("Telegram bot process is running"))
}

// /health gives configured admins the same checks without public disclosure.
func HealthCommand(bot *gotgbot.Bot,ctx *ext.Context)error {
 if !privateAdminOnly(bot,ctx){return nil}
 pctx,cancel:=context.WithTimeout(context.Background(),5*time.Second)
 defer cancel()
 s:=currentHealth(pctx)
 icon:=func(ok bool)string{if ok{return "✅"};return "❌"}
 msg:=fmt.Sprintf("🩺 Bot health\n%s Telegram polling started\n%s Telegram API reachable\n%s MongoDB connected",
  icon(s.Started),icon(s.Telegram),icon(s.Mongo))
 if !s.Healthy(){msg+="\n⚠️ A health check failed. Check Render logs."}
 _,err:=bot.SendMessage(ctx.EffectiveChat.Id,msg,nil)
 return err
}
