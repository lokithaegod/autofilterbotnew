package mongo

import (
 "context"
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
