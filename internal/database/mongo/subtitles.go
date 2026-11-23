package mongo

import (
 "regexp"
 "strings"

 "github.com/Jisin0/autofilterbot/internal/subtitles"
 "go.mongodb.org/mongo-driver/bson"
 "go.mongodb.org/mongo-driver/mongo"
 "go.mongodb.org/mongo-driver/mongo/options"
)

func (c *Client) SaveSubtitle(entry *subtitles.Entry) error {
 if entry==nil || entry.UniqueID=="" || entry.FileID=="" || entry.TitleKey=="" || entry.Year=="" {
  return mongo.ErrNoDocuments
 }
 _,err:=c.subtitleCollection.ReplaceOne(c.ctx,bson.M{"_id":entry.UniqueID},entry,options.Replace().SetUpsert(true))
 return err
}

func (c *Client) SearchSubtitles(key,year string)([]*subtitles.Entry,error){
 key=subtitles.Normalize(key)
 if key=="" {return nil,nil}
 filter:=bson.M{"title_key":bson.M{"$regex":regexp.QuoteMeta(key)}}
 if year!="" {filter["year"]=year;filter["title_key"]=key}
 cursor,err:=c.subtitleCollection.Find(c.ctx,filter,options.Find().SetLimit(15).SetSort(bson.D{{Key:"file_name",Value:1}}))
 if err!=nil{return nil,err}
 defer cursor.Close(c.ctx)
 var result []*subtitles.Entry
 for cursor.Next(c.ctx){
  var entry subtitles.Entry
  if err:=cursor.Decode(&entry);err!=nil{return nil,err}
  result=append(result,&entry)
 }
 return result,cursor.Err()
}

func (c *Client) SubtitleExists(key,year string)(bool,error){
 key=subtitles.Normalize(key)
 if key==""||strings.TrimSpace(year)==""{return false,nil}
 err:=c.subtitleCollection.FindOne(c.ctx,bson.M{"title_key":key,"year":year},options.FindOne().SetProjection(bson.M{"_id":1})).Err()
 if err==mongo.ErrNoDocuments{return false,nil}
 return err==nil,err
}

func (c *Client) GetSubtitle(id string)(*subtitles.Entry,error){
 var entry subtitles.Entry
 err:=c.subtitleCollection.FindOne(c.ctx,bson.M{"_id":id}).Decode(&entry)
 if err!=nil{return nil,err}
 return &entry,nil
}
