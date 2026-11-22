package core

import (
 "context"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "net/http"
 "net/url"
 "os"
 "strconv"
 "strings"
 "time"

 "github.com/PaulSonOfLars/gotgbot/v2"
 "github.com/PaulSonOfLars/gotgbot/v2/ext"
 "go.uber.org/zap"
)

const adsgramAPI="https://api.adsgram.ai/advbot"
const adsgramSearchInterval=10
const adsgramDailyCooldown=24*time.Hour

// Ads remain disabled until the owner separately authorizes production use.
// Never silently turn on advertisements just because provider keys are present.
func adsGramEnabled()bool {
 return os.Getenv("ADSGRAM_ENABLED")=="true" &&
  os.Getenv("ADSGRAM_OWNER_APPROVED")=="true" &&
  strings.TrimSpace(os.Getenv("ADSGRAM_BLOCK_ID"))!="" &&
  os.Getenv("ADSGRAM_TOKEN")!="" &&
  os.Getenv("ADSGRAM_PACING_SALT")!=""
}

type adsgramCreative struct {
 TextHTML string `json:"text_html"`
 ClickURL string `json:"click_url"`
 ButtonName string `json:"button_name"`
 ImageURL string `json:"image_url"`
 RewardButtonName string `json:"button_reward_name"`
 RewardURL string `json:"reward_url"`
}

func adsGramSafeURL(raw string)bool{
 u,err:=url.Parse(raw)
 return err==nil&&u.Scheme=="https"&&u.User==nil&&u.Port()==""&&strings.EqualFold(u.Hostname(),"api.adsgram.ai")&&strings.HasPrefix(u.EscapedPath(),"/short/")
}

func cleanAdCreative(ad adsgramCreative)(adsgramCreative,error){
 if len(ad.TextHTML)==0||len(ad.TextHTML)>3500{return ad,errors.New("invalid ad text")}
 if !adsGramSafeURL(ad.ClickURL)||!adsGramSafeURL(ad.RewardURL) {return ad,errors.New("invalid ad destination")}
 if len(ad.ButtonName)>55||len(ad.RewardButtonName)>55{return ad,errors.New("invalid ad button")}
 if ad.ButtonName==""{ad.ButtonName="View sponsor"}
 if ad.RewardButtonName==""{ad.RewardButtonName="Claim reward"}
 return ad,nil
}

func fetchAdsGramAd(ctx context.Context,userID int64,language string,client *http.Client)(adsgramCreative,error){
 return fetchAdsGramAdFrom(ctx,userID,language,client,adsgramAPI)
}

func fetchAdsGramAdFrom(ctx context.Context,userID int64,language string,client *http.Client,endpoint string)(adsgramCreative,error){
 var empty adsgramCreative
 blockID:=strings.TrimPrefix(os.Getenv("ADSGRAM_BLOCK_ID"),"bot-")
 if _,err:=strconv.ParseUint(blockID,10,64);err!=nil{return empty,errors.New("invalid AdsGram block ID")}
 if len(language)<2||len(language)>8{language="en"}
 params:=url.Values{}
 params.Set("tgid",strconv.FormatInt(userID,10))
 params.Set("blockid",blockID)
 params.Set("language",language)
 params.Set("token",os.Getenv("ADSGRAM_TOKEN"))
 base,err:=url.Parse(endpoint)
 if err!=nil{return empty,err}
 existing:=base.Query()
 for k,vs:=range params {existing.Set(k,vs[0])}
 base.RawQuery=existing.Encode()
 req,err:=http.NewRequestWithContext(ctx,http.MethodGet,base.String(),nil)
 if err!=nil{return empty,err}
 resp,err:=client.Do(req)
 if err!=nil{return empty,err}
 defer resp.Body.Close()
 if resp.StatusCode==http.StatusNoContent{return empty,nil}
 if resp.StatusCode!=http.StatusOK{return empty,fmt.Errorf("provider returned HTTP %d",resp.StatusCode)}
 var ad adsgramCreative
 if err=json.NewDecoder(io.LimitReader(resp.Body,32<<10)).Decode(&ad);err!=nil{return empty,err}
 return cleanAdCreative(ad)
}

// maybeSendAdsGram is called ONLY after an ordinary, successful search-result
// message has been delivered. It never blocks access to files or mutates bot
// settings, and sends ads only to the requesting user in a private Telegram chat.
func maybeSendAdsGram(bot *gotgbot.Bot,ctx *ext.Context,userID int64){
 if !adsGramEnabled()||ctx==nil||ctx.Message==nil||ctx.Message.Chat.Type!="private"||userID<=0{return}
 pacing,ok:=_app.DB.(interface{AdPacingStep(int64,string)(bool,error)})
 if !ok{return}
 eligible,err:=pacing.AdPacingStep(userID,os.Getenv("ADSGRAM_PACING_SALT"))
 if err!=nil {
  _app.Log.Warn("adsgram: pacing unavailable",zap.Error(err));return
 }
 if !eligible{return}
 requestCtx,cancel:=context.WithTimeout(context.Background(),5*time.Second)
 defer cancel()
 client:=&http.Client{Timeout:4*time.Second,CheckRedirect:func(_ *http.Request,_ []*http.Request)error{return http.ErrUseLastResponse}}
 ad,err:=fetchAdsGramAd(requestCtx,userID,ctx.Message.From.LanguageCode,client)
 if err!=nil{_app.Log.Warn("adsgram: ad request unsuccessful",zap.Error(err));return}
 if ad.ClickURL==""{return}
 // Use text-only ads: image_url is optional and image fetch is not necessary.
 // Provider HTML is explicitly required by AdsGram's Bot Ad Block API.
 keyboard:=gotgbot.InlineKeyboardMarkup{InlineKeyboard:[][]gotgbot.InlineKeyboardButton{
  {{Text:ad.ButtonName,Url:ad.ClickURL}},
  {{Text:ad.RewardButtonName,Url:ad.RewardURL}},
 }}
 _,err=bot.SendMessage(userID,"<b>Sponsored message</b>\n"+ad.TextHTML,&gotgbot.SendMessageOpts{
  ParseMode:gotgbot.ParseModeHTML,
  ProtectContent:true,
  ReplyMarkup:keyboard,
 })
 if err!=nil{_app.Log.Warn("adsgram: sponsored message delivery failed",zap.Error(err))}
}

// AdsGramRewardHTTP is deliberately fail-closed until an authenticated reward
// verification protocol and an owner-approved benefit have been specified.
// A bare GET, even with a Telegram user ID, is NOT proof of ad completion.
func AdsGramRewardHTTP(w http.ResponseWriter,r *http.Request){
 w.Header().Set("Cache-Control","no-store")
 w.Header().Set("Content-Type","text/plain; charset=utf-8")
 if !adsGramEnabled(){http.NotFound(w,r);return}
 if r.Method!=http.MethodGet {w.WriteHeader(http.StatusMethodNotAllowed);return}
 w.WriteHeader(http.StatusNotImplemented)
 _,_=w.Write([]byte("Reward claiming is not configured; no reward was issued"))
}
