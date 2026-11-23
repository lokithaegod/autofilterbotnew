package mongo

import (
 "context"
 "errors"
 "strings"
 "time"

 "github.com/Jisin0/autofilterbot/internal/model"
 "go.mongodb.org/mongo-driver/bson"
 "go.mongodb.org/mongo-driver/mongo"
 "go.mongodb.org/mongo-driver/mongo/options"
)

// KeywordReply is a safe text-only reply. It never evaluates user-supplied code.
type KeywordReply struct {
 Scope string `bson:"scope"`
 ChatID int64 `bson:"chat_id"`
 Keyword string `bson:"keyword"`
 Reply string `bson:"reply"`
}
func (c *Client) PutKeywordReply(f KeywordReply)error {
 ctx,cancel:=context.WithTimeout(context.Background(),8*time.Second);defer cancel()
 _,err:=c.db.Collection("KeywordReplies").UpdateOne(ctx,
  bson.M{"scope":f.Scope,"chat_id":f.ChatID,"keyword":f.Keyword},
  bson.M{"$set":f},options.Update().SetUpsert(true))
 return err
}
func (c *Client) DeleteKeywordReply(scope string,chatID int64,keyword string)(bool,error){
 ctx,cancel:=context.WithTimeout(context.Background(),8*time.Second);defer cancel()
 result,err:=c.db.Collection("KeywordReplies").DeleteOne(ctx,bson.M{"scope":scope,"chat_id":chatID,"keyword":keyword})
 return result!=nil&&result.DeletedCount>0,err
}
func (c *Client) ListKeywordReplies(scope string,chatID int64)([]KeywordReply,error){
 ctx,cancel:=context.WithTimeout(context.Background(),8*time.Second);defer cancel()
 cur,err:=c.db.Collection("KeywordReplies").Find(ctx,bson.M{"scope":scope,"chat_id":chatID},options.Find().SetLimit(100).SetSort(bson.D{{Key:"keyword",Value:1}}))
 if err!=nil{return nil,err};defer cur.Close(ctx)
 var out []KeywordReply
 for cur.Next(ctx){var r KeywordReply;if err:=cur.Decode(&r);err!=nil{return nil,err};out=append(out,r)}
 return out,cur.Err()
}

// Welcome photo settings are isolated from the existing welcome message config.
func (c *Client) AddWelcomePhoto(fileID string)error{
 ctx,cancel:=context.WithTimeout(context.Background(),8*time.Second);defer cancel()
 _,err:=c.db.Collection("WelcomePhotos").UpdateOne(ctx,bson.M{"_id":fileID},bson.M{"$set":bson.M{"enabled":true}},options.Update().SetUpsert(true));return err
}
func (c *Client) ListWelcomePhotos()([]string,error){
 ctx,cancel:=context.WithTimeout(context.Background(),8*time.Second);defer cancel()
 cur,err:=c.db.Collection("WelcomePhotos").Find(ctx,bson.M{"enabled":true},options.Find().SetLimit(30))
 if err!=nil{return nil,err};defer cur.Close(ctx)
 var result []string
 for cur.Next(ctx){var doc struct{ID string `bson:"_id"`};if err:=cur.Decode(&doc);err!=nil{return nil,err};if doc.ID!=""{result=append(result,doc.ID)}}
 return result,cur.Err()
}
func (c *Client) RemoveWelcomePhoto(fileID string)(bool,error){
 ctx,cancel:=context.WithTimeout(context.Background(),8*time.Second);defer cancel()
 res,err:=c.db.Collection("WelcomePhotos").DeleteOne(ctx,bson.M{"_id":fileID});return res!=nil&&res.DeletedCount>0,err
}

// DuplicateCandidates obtains a bounded, read-only snapshot of original indexed
// files across all configured file collections. This never modifies file IDs.
func (c *Client) DuplicateCandidates(limit int64)([]model.File,bool,error){
 if limit<2||limit>5000{limit=1500}
 ctx,cancel:=context.WithTimeout(context.Background(),18*time.Second);defer cancel()
 result:=make([]model.File,0,limit)
 truncated:=false
 for _,coll:=range c.fileCollection.allCollections{
  remaining:=limit-int64(len(result))
  if remaining<=0{truncated=true;break}
  cursor,err:=coll.Find(ctx,bson.M{},options.Find().SetSort(bson.D{{Key:"time",Value:-1}}).SetLimit(remaining+1))
  if err!=nil{return nil,false,err}
  for cursor.Next(ctx){
   var f model.File
   if err:=cursor.Decode(&f);err!=nil{cursor.Close(ctx);return nil,false,err}
   if int64(len(result))>=limit {truncated=true;break}
   if strings.TrimSpace(f.UniqueId)!="" {result=append(result,f)}
  }
  err=cursor.Err();cursor.Close(ctx)
  if err!=nil&&!errors.Is(err,mongo.ErrNoDocuments){return nil,false,err}
 }
 return result,truncated,nil
}
