package mongo

import (
 "context"
 "crypto/hmac"
 "crypto/sha256"
 "encoding/hex"
 "strconv"
 "sync"
 "time"

 "go.mongodb.org/mongo-driver/bson"
 "go.mongodb.org/mongo-driver/mongo"
 "go.mongodb.org/mongo-driver/mongo/options"
)

// AdPacingStep records only a keyed, pseudonymous user identifier and a
// successful search count. It is never called when advertisements are disabled.
// It reserves the at-most-once-per-day opportunity atomically across bot restarts.
func (c *Client) AdPacingStep(userID int64, salt string) (bool,error) {
 if salt=="" {return false,nil}
 mac:=hmac.New(sha256.New,[]byte(salt))
 _,_=mac.Write([]byte(strconv.FormatInt(userID,10)))
 key:=hex.EncodeToString(mac.Sum(nil))
 ctx,cancel:=context.WithTimeout(context.Background(),3*time.Second)
 defer cancel()
 now:=time.Now().UTC()
 col:=c.db.Collection("AdsGramPacing")
 // A 90-day TTL removes stale anonymous counters. The index is created only
 // once for a running bot, and only when ad serving is explicitly enabled.
 if err:=ensureAdsPacingIndex(ctx,col);err!=nil{return false,err}
 var state struct{Searches int64 `bson:"searches"`; LastSent time.Time `bson:"last_sent"`}
 err:=col.FindOneAndUpdate(ctx,
  bson.M{"_id":key},
  bson.M{"$inc":bson.M{"searches":1},"$set":bson.M{"expires_at":now.Add(90*24*time.Hour)},"$setOnInsert":bson.M{"last_sent":time.Time{}}},
  options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)).Decode(&state)
 if err!=nil{return false,err}
 if state.Searches<10 || (!state.LastSent.IsZero()&&now.Sub(state.LastSent)<24*time.Hour){return false,nil}
 // Reserve before sending to avoid duplicates from simultaneous searches.
 res,err:=col.UpdateOne(ctx,
  bson.M{"_id":key,"searches":bson.M{"$gte":10},"last_sent":bson.M{"$lt":now.Add(-24*time.Hour)}},
  bson.M{"$set":bson.M{"last_sent":now,"searches":0}})
 if err!=nil{return false,err}
 return res.ModifiedCount==1,nil
}

var adsPacingIndex=struct{sync.Mutex; ready bool}{}
func ensureAdsPacingIndex(ctx context.Context,col *mongo.Collection)error{
 adsPacingIndex.Lock()
 defer adsPacingIndex.Unlock()
 if adsPacingIndex.ready{return nil}
 _,err:=col.Indexes().CreateOne(ctx,mongo.IndexModel{
  Keys:bson.D{{Key:"expires_at",Value:1}},
  Options:options.Index().SetExpireAfterSeconds(0),
 })
 if err==nil{adsPacingIndex.ready=true}
 return err
}
