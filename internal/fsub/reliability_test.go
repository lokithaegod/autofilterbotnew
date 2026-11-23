package fsub

import (
 "strings"
 "testing"
)
func TestRetryDeepLinkPreservesToken(t *testing.T) {
 cases:=[]struct{args []string;want string}{
  {[]string{"start","Zl9maWxl"}, "https://t.me/AutoFilterMGbot?start=Retry_Zl9maWxl"},
  {[]string{"start","Retry_Zl9maWxl"},"https://t.me/AutoFilterMGbot?start=Retry_Zl9maWxl"},
  {[]string{"start"},"https://t.me/AutoFilterMGbot?start"},
  {[]string{"start",""},"https://t.me/AutoFilterMGbot?start"},
 }
 for _,tc:=range cases{
  got:=RetryDeepLink("AutoFilterMGbot",tc.args)
  if got!=tc.want{t.Errorf("args %v: got %q want %q",tc.args,got,tc.want)}
 }
 long:=strings.Repeat("A",63)
 if got:=RetryDeepLink("bot",[]string{"start",long});got!="https://t.me/bot?start="+long{
  t.Fatal("oversize retry marker should fall back to existing token")
 }
}

func TestProcessLocalStatsCountEvents(t *testing.T) {
 initial:=Snapshot()
 RecordRetry()
 RecordJoinRequest()
 now:=Snapshot()
 if now.Retries!=initial.Retries+1 || now.JoinRequests!=initial.JoinRequests+1 {
  t.Fatal("retry/join request statistics did not increase")
 }
}
