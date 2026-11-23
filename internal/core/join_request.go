package core

import (
 "github.com/Jisin0/autofilterbot/internal/fsub"
 "github.com/PaulSonOfLars/gotgbot/v2"
 "github.com/PaulSonOfLars/gotgbot/v2/ext"
)

// HandleJoinRequest records an anonymous event for configured ForceSub channels.
// It must not record historical join requests as membership entitlements.
// Telegram membership is checked live when the user requests a file.
func HandleJoinRequest(bot *gotgbot.Bot, ctx *ext.Context) error {
 if ctx.ChatJoinRequest==nil{return nil}
 for _,channel:=range _app.Config.GetFsubChannels(){
  if channel.ID==ctx.ChatJoinRequest.Chat.Id {
   fsub.RecordJoinRequest()
   break
  }
 }
 return nil
}
