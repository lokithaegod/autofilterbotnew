package mongo

import (
 "context"
 "fmt"
 "regexp"
 "strings"
 "time"

 "github.com/Jisin0/autofilterbot/internal/compatfinder"
 "github.com/Jisin0/autofilterbot/internal/model"
 "go.mongodb.org/mongo-driver/bson"
 "go.mongodb.org/mongo-driver/mongo/options"
)

// FindCompatibilityCandidates searches every existing movie file collection
// without using the ordinary SearchFiles 500-result limit. No schema writes.
func (c *Client) FindCompatibilityCandidates(subtitleName string)([]*model.File,error){
 title,year,ok:=compatfinder.Identity(subtitleName)
 if !ok{return nil,fmt.Errorf("subtitle must have a recognizable title and release year")}
 terms:=strings.Fields(title)
 if len(terms)==0 || len(terms)>20{return nil,fmt.Errorf("invalid subtitle title")}
 for i,word:=range terms {terms[i]=regexp.QuoteMeta(word)}
 pattern:="(?i)(^|[^a-z0-9])"+strings.Join(terms,"[^a-z0-9]*")+"([^a-z0-9]|$)"
 yearPattern:="(^|[^0-9])"+regexp.QuoteMeta(year)+"([^0-9]|$)"
 filter:=bson.M{"$and":[]bson.M{
  {"file_name":bson.M{"$regex":pattern}},
  {"file_name":bson.M{"$regex":yearPattern}},
  {"file_type":bson.M{"$in":[]string{"video","document"}}},
 }}
 ctx,cancel:=context.WithTimeout(context.Background(),60*time.Second)
 defer cancel()
 results:=make([]*model.File,0,64)
 seen:=make(map[string]bool)
 const maxCandidates=10000
 for _,collection:=range c.fileCollection.allCollections{
  cursor,err:=collection.Find(ctx,filter,options.Find().SetBatchSize(200).SetProjection(bson.M{
   "_id":1,"file_id":1,"file_name":1,"file_type":1,"file_size":1,"chat_id":1,
  }))
  if err!=nil{return nil,err}
  for cursor.Next(ctx){
   var item model.File
   if err:=cursor.Decode(&item);err!=nil{cursor.Close(ctx);return nil,err}
   // Ambiguous old document records can be candidates, but never verified
   // without actual media and a successful audio check.
   if item.UniqueId==""||seen[item.UniqueId]{continue}
   seen[item.UniqueId]=true
   results=append(results,&item)
   if len(results)>maxCandidates{
    cursor.Close(ctx)
    return nil,fmt.Errorf("too many candidates; search was not complete")
   }
  }
  err=cursor.Err()
  cursor.Close(ctx)
  if err!=nil{return nil,err}
 }
 return results,nil
}
