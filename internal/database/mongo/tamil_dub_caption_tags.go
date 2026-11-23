package mongo

import (
 "context"
 "time"

 "github.com/Jisin0/autofilterbot/internal/functions"
 "go.mongodb.org/mongo-driver/bson"
 "go.mongodb.org/mongo-driver/mongo"
 "go.mongodb.org/mongo-driver/mongo/options"
)

// TamilDubCaptionTags is a separate classification overlay, not a file
// collection. No existing movie, file ID or Malayalam caption is changed.
func (c *Client) SetTamilDubCaptionTag(id string,enabled bool)error{
 if id==""{return mongo.ErrNoDocuments}
 ctx,cancel:=context.WithTimeout(context.Background(),8*time.Second)
 defer cancel()
 col:=c.db.Collection("TamilDubCaptionTags")
 if !enabled{
  _,err:=col.DeleteOne(ctx,bson.M{"_id":id})
  return err
 }
 _,err:=col.UpdateOne(ctx,bson.M{"_id":id},
  bson.M{"$set":bson.M{"manually_tagged":true}},
  options.Update().SetUpsert(true))
 return err
}

func (c *Client) FindTamilDubCaptionTags(ids []string)(map[string]bool,error){
 tags:=make(map[string]bool)
 if len(ids)==0{return tags,nil}
 ctx,cancel:=context.WithTimeout(context.Background(),8*time.Second)
 defer cancel()
 rows,err:=c.db.Collection("TamilDubCaptionTags").Find(ctx,
  bson.M{"_id":bson.M{"$in":ids},"manually_tagged":true},
  options.Find().SetProjection(bson.M{"_id":1}))
 if err!=nil{return nil,err}
 defer rows.Close(ctx)
 for rows.Next(ctx){
  var row struct{ID string `bson:"_id"`}
  if err:=rows.Decode(&row);err!=nil{return nil,err}
  tags[row.ID]=true
 }
 return tags,rows.Err()
}

// Match the original indexed movie by exact file size and either original or
// historically sanitized filename, as with Malayalam caption tags.
func (c *Client) SetTamilDubCaptionTagForMedia(uniqueID,filename string,size int64,enabled bool)(int,error){
 if uniqueID==""{return 0,mongo.ErrNoDocuments}
 linked:=map[string]bool{uniqueID:true}
 if filename!=""&&size>0{
  variations:=[]string{filename,functions.RemoveSymbols(functions.RemoveExtension(filename))}
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
 for id:=range linked{
  if err:=c.SetTamilDubCaptionTag(id,enabled);err!=nil{return count,err}
  count++
 }
 return count,nil
}
