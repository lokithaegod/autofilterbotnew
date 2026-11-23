package mongo

import (
 "context"

 "github.com/Jisin0/autofilterbot/internal/functions"
 "time"

 "go.mongodb.org/mongo-driver/bson"
 "go.mongodb.org/mongo-driver/mongo"
 "go.mongodb.org/mongo-driver/mongo/options"
)

// SetMalayalamCaptionTag records a MANUAL caption claim only. This never
// modifies existing Files documents or counts as verified timing evidence.
// Repeating a channel post with the same file unique ID is idempotent.
func (c *Client) SetMalayalamCaptionTag(uniqueID string, enabled bool) error {
 if uniqueID==""{return mongo.ErrNoDocuments}
 ctx,cancel:=context.WithTimeout(context.Background(),8*time.Second)
 defer cancel()
 coll:=c.db.Collection("MalayalamCaptionTags")
 if !enabled{
  _,err:=coll.DeleteOne(ctx,bson.M{"_id":uniqueID})
  return err
 }
 _,err:=coll.UpdateOne(ctx,bson.M{"_id":uniqueID},
  bson.M{"$set":bson.M{"manually_tagged":true}},
  options.Update().SetUpsert(true))
 return err
}

// FindMalayalamCaptionTags fetches tags for a bounded movie-result page.
// Missing IDs are explicitly untagged, not automatically compatible.
func (c *Client) FindMalayalamCaptionTags(uniqueIDs []string)(map[string]bool,error){
 tagged:=make(map[string]bool)
 if len(uniqueIDs)==0{return tagged,nil}
 ctx,cancel:=context.WithTimeout(context.Background(),8*time.Second)
 defer cancel()
 cursor,err:=c.db.Collection("MalayalamCaptionTags").Find(ctx,
  bson.M{"_id":bson.M{"$in":uniqueIDs},"manually_tagged":true},
  options.Find().SetProjection(bson.M{"_id":1}))
 if err!=nil{return nil,err}
 defer cursor.Close(ctx)
 for cursor.Next(ctx){
  var doc struct{ID string `bson:"_id"`}
  if err:=cursor.Decode(&doc);err!=nil{return nil,err}
  tagged[doc.ID]=true
 }
 return tagged,cursor.Err()
}

// SetMalayalamCaptionTagForMedia bridges Telegram Bot API file_unique_id and
// older database file IDs (which can differ for independently indexed copies).
// No original file document is modified. Matching requires both the exact
// filename (raw or historical sanitized form) AND the exact byte size.
func (c *Client) SetMalayalamCaptionTagForMedia(uniqueID, filename string, size int64, enabled bool) (int,error) {
 if uniqueID==""{return 0,mongo.ErrNoDocuments}
 linked:=map[string]bool{uniqueID:true}
 if filename!=""&&size>0{
  variations:=[]string{filename,functions.RemoveSymbols(functions.RemoveExtension(filename))}
  // Old file rows can contain either a full file extension or sanitized name.
  filter:=bson.M{"file_name":bson.M{"$in":variations},"file_size":size}
  ctx,cancel:=context.WithTimeout(context.Background(),12*time.Second)
  defer cancel()
  for _,coll:=range c.fileCollection.allCollections{
   cursor,err:=coll.Find(ctx,filter,options.Find().SetProjection(bson.M{"_id":1}).SetLimit(100))
   if err!=nil{return 0,err}
   for cursor.Next(ctx){
    var row struct{ID string `bson:"_id"`}
    if err:=cursor.Decode(&row);err!=nil{cursor.Close(ctx);return 0,err}
    if row.ID!=""{linked[row.ID]=true}
   }
   err=cursor.Err()
   cursor.Close(ctx)
   if err!=nil{return 0,err}
  }
 }
 count:=0
 for candidate:=range linked{
  if err:=c.SetMalayalamCaptionTag(candidate,enabled);err!=nil{return count,err}
  count++
 }
 return count,nil
}

