package core

import (
 "context"
 "fmt"
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"
)

func TestAdsGramOffByDefault(t *testing.T){
 t.Setenv("ADSGRAM_ENABLED","")
 t.Setenv("ADSGRAM_OWNER_APPROVED","")
 t.Setenv("ADSGRAM_BLOCK_ID","")
 t.Setenv("ADSGRAM_TOKEN","")
 t.Setenv("ADSGRAM_PACING_SALT","")
 if adsGramEnabled(){t.Fatal("ads must default to OFF")}
 // Even valid credentials do not bypass separate owner consent.
 t.Setenv("ADSGRAM_BLOCK_ID","bot-123")
 t.Setenv("ADSGRAM_TOKEN","private-test-key")
 t.Setenv("ADSGRAM_PACING_SALT","test-salt")
 t.Setenv("ADSGRAM_ENABLED","true")
 if adsGramEnabled(){t.Fatal("ads must remain off without final owner approval")}
 t.Setenv("ADSGRAM_OWNER_APPROVED","true")
 if !adsGramEnabled(){t.Fatal("fully approved test configuration didn't activate")}
}

func TestRewardHandlerNeverGrantsUnverifiedReward(t *testing.T){
 t.Setenv("ADSGRAM_ENABLED","false")
 req:=httptest.NewRequest(http.MethodGet,"https://example.test/ads/reward?userId=123",nil)
 resp:=httptest.NewRecorder()
 AdsGramRewardHTTP(resp,req)
 if resp.Code!=http.StatusNotFound{t.Fatalf("off status: %d",resp.Code)}
 t.Setenv("ADSGRAM_ENABLED","true")
 t.Setenv("ADSGRAM_OWNER_APPROVED","true")
 t.Setenv("ADSGRAM_BLOCK_ID","bot-123")
 t.Setenv("ADSGRAM_TOKEN","test-token")
 t.Setenv("ADSGRAM_PACING_SALT","test-salt")
 resp=httptest.NewRecorder()
 AdsGramRewardHTTP(resp,req)
 if resp.Code!=http.StatusNotImplemented{t.Fatalf("unverified callback status: %d",resp.Code)}
 if !strings.Contains(resp.Body.String(),"no reward was issued"){t.Fatal("unverified callback was not rejected")}
}

func TestAdsGramCreativeValidation(t *testing.T){
 good:=adsgramCreative{TextHTML:"<b>Example</b>",ClickURL:"https://api.adsgram.ai/short/abcd",RewardURL:"https://api.adsgram.ai/short/reward"}
 if _,err:=cleanAdCreative(good);err!=nil{t.Fatal(err)}
 bad:=[]string{
  "http://api.adsgram.ai/short/click",
  "https://evil.example/short/click",
  "https://api.adsgram.ai.evil.example/short/click",
  "https://api.adsgram.ai:8443/short/click",
  "https://adsgram.ai/short/click",
 }
 for _,link:=range bad{
  other:=good;other.ClickURL=link
  if _,err:=cleanAdCreative(other);err==nil{t.Errorf("accepted unsafe URL: %s",link)}
 }
}

func TestAdsGramAPIAndEmptyAd(t *testing.T){
 t.Setenv("ADSGRAM_BLOCK_ID","bot-123")
 t.Setenv("ADSGRAM_TOKEN","not-a-real-token")
 server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  if got:=r.URL.Query().Get("tgid");got!="456"{t.Errorf("unexpected user ID %q",got)}
  if got:=r.URL.Query().Get("blockid");got!="123"{t.Errorf("unexpected block ID %q",got)}
  if got:=r.URL.Query().Get("token");got!="not-a-real-token"{t.Error("token not forwarded to provider")}
  if r.URL.Query().Get("language")!="en"{t.Error("invalid language not normalized")}
  if r.URL.Query().Get("empty")=="1"{w.WriteHeader(http.StatusNoContent);return}
  w.Header().Set("Content-Type","application/json")
  _,_=fmt.Fprint(w,`{"text_html":"<b>Ad</b>","click_url":"https://api.adsgram.ai/short/click","reward_url":"https://api.adsgram.ai/short/reward","button_name":"Visit sponsor","button_reward_name":"Claim reward!"}`)
 }))
 defer server.Close()
 ad,err:=fetchAdsGramAdFrom(context.Background(),456,"",server.Client(),server.URL)
 if err!=nil{t.Fatal(err)}
 if ad.ButtonName!="Visit sponsor"{t.Fatalf("ad not decoded: %+v",ad)}
 ad,err=fetchAdsGramAdFrom(context.Background(),456,"",server.Client(),server.URL+"?empty=1")
 // Caller should treat 204 as no ad, not an error.
 if err!=nil||ad.ClickURL!=""{t.Fatalf("expected no-ad result: %+v %v",ad,err)}
}
