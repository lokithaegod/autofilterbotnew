package main

import (
 "log"
 "net/http"
 "os"

 "github.com/Jisin0/autofilterbot/cmd"
 "github.com/Jisin0/autofilterbot/internal/core"
)

// Keep the existing public URL compatible with UptimeRobot and GitHub checks,
// but return 503 if the bot, database, or Telegram API are not healthy.
func main() {
 if port:=os.Getenv("PORT");port!="" {
  mux:=http.NewServeMux()
  mux.HandleFunc("/",core.HealthHTTP)
  mux.HandleFunc("/healthz",core.HealthHTTP)
  go func(){
   log.Printf("Starting HTTP listener on port %s",port)
   if err:=http.ListenAndServe(":"+port,mux);err!=nil {log.Printf("Health listener stopped: %v",err)}
  }()
 }
 cmd.Execute()
}
