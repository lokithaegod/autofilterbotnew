package mongo

import (
 "context"
 "time"

 "go.mongodb.org/mongo-driver/bson"
 "go.mongodb.org/mongo-driver/mongo/options"
)

// SuggestionFilenames lists indexed filenames for typo correction without
// exposing file IDs, user data, or mutating the existing file collection.
// Bounded by a deadline and an upper limit to protect Render Free resources.
func (c *Client) SuggestionFilenames() ([]string,error) {
 ctx,cancel:=context.WithTimeout(context.Background(),8*time.Second)
 defer cancel()
 cur,err:=c.db.Collection("Files").Find(ctx,bson.M{},
  options.Find().SetProjection(bson.M{"file_name":1,"_id":0}).SetLimit(12000))
 if err!=nil{return nil,err}
 defer cur.Close(ctx)
 var rows []struct {Name string `bson:"file_name"`}
 if err:=cur.All(ctx,&rows);err!=nil{return nil,err}
 out:=make([]string,0,len(rows))
 for _,row:=range rows{if row.Name!="" {out=append(out,row.Name)}}
 return out,nil
}
