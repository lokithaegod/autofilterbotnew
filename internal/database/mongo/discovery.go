package mongo

import (
 "context"
 "regexp"
 "strings"
 "time"

 "github.com/Jisin0/autofilterbot/internal/database"
 "go.mongodb.org/mongo-driver/bson"
 "go.mongodb.org/mongo-driver/mongo"
 "errors"
 "go.mongodb.org/mongo-driver/mongo/options"
)

// CatalogFilter powers discovery and filtered searches without changing old file documents.
type CatalogFilter struct {
 Query string
 Year string
 Quality string
 Language string
 Type string
 Sort string
 MinSize int64
 MaxSize int64
 Category string
 MediaOnly bool
}

func (c *Client) CatalogCursor(ctx context.Context, f CatalogFilter, limit int64) (database.Cursor,error) {
 terms:=bson.A{}
 if f.Query!="" || f.Year!="" {
  if f.Query!="" { terms=append(terms,movieSearchFilter(f.Query,f.Year)) } else {
   terms=append(terms,bson.M{"file_name":bson.M{"$regex":filenameSearchPattern(f.Year)}})
  }
 }
 if f.Quality!="" && f.Quality!="all" {
  q:=f.Quality
  if q=="2160p" {q="(2160p|4k)"} else {q=regexp.QuoteMeta(q)}
  terms=append(terms,bson.M{"file_name":bson.M{"$regex":`(^|[^a-z0-9])`+q+`([^a-z0-9]|$)`,"$options":"i"}})
 }
 if f.Type!="" && f.Type!="all" {terms=append(terms,bson.M{"file_type":f.Type})}
 size:=bson.M{}
 if f.MinSize>0 {size["$gte"]=f.MinSize}
 if f.MaxSize>0 {size["$lte"]=f.MaxSize}
 if len(size)>0 {terms=append(terms,bson.M{"file_size":size})}
 switch f.Category {
 case "movies": terms=append(terms,bson.M{"file_name":bson.M{"$regex":`(19|20)[0-9]{2}`}})
 case "series":terms=append(terms,bson.M{"file_name":bson.M{"$regex":`S[0-9]{1,2}E[0-9]{1,3}`,"$options":"i"}})
 case "documents":terms=append(terms,bson.M{"file_name":bson.M{"$regex":`\.(pdf|epub|docx?|pptx?|txt)$`,"$options":"i"}})
 case "subs":
  pattern:=bson.M{"$regex":`(?i)\.(srt|ass|ssa|vtt|sub)$`}
  terms=append(terms,bson.M{"$or":bson.A{bson.M{"file_name":pattern},bson.M{"original_file_name":pattern}}})
 case "latest": // no additional filter
 }
 filter:=bson.M{}
 if len(terms)>0 {filter["$and"]=terms}
 order:=bson.D{{Key:"time",Value:-1}}
 if f.Sort=="smallest" {order=bson.D{{Key:"file_size",Value:1},{Key:"time",Value:-1}}}
 if f.Sort=="largest" {order=bson.D{{Key:"file_size",Value:-1},{Key:"time",Value:-1}}}
 opts:=options.Find().SetSort(order)
 if !f.MediaOnly && (f.Language=="" || f.Language=="all") {opts.SetLimit(limit)}
 return c.fileCollection.Find(ctx,filter,opts)
}

type SearchMetric struct {
 Query string `bson:"_id"`
 Count int64 `bson:"count"`
}

func (c *Client) LogSearch(query string) error {
 query=strings.TrimSpace(strings.ToLower(query))
 if query==""||len(query)>120 {return nil}
 // Aggregate counts only; no user IDs or private search histories are stored.
 _,err:=c.db.Collection("SearchMetrics").UpdateOne(c.ctx,bson.M{"_id":query},
 bson.M{"$inc":bson.M{"count":1},"$set":bson.M{"updated_at":time.Now()}},options.Update().SetUpsert(true))
 return err
}

func (c *Client) TopSearches(limit int64)([]SearchMetric,error){
 cur,err:=c.db.Collection("SearchMetrics").Find(c.ctx,bson.M{},options.Find().SetSort(bson.D{{Key:"count",Value:-1}}).SetLimit(limit))
 if err!=nil{return nil,err}
 defer cur.Close(c.ctx)
 var values []SearchMetric
 err=cur.All(c.ctx,&values)
 return values,err
}

func (c *Client) SetUserBlocked(id int64,blocked bool)error {
 _,err:=c.db.Collection("BlockedUsers").UpdateOne(c.ctx,bson.M{"_id":id},bson.M{"$set":bson.M{"blocked":blocked}},options.Update().SetUpsert(true))
 return err
}
func (c *Client) IsUserBlocked(id int64)(bool,error){
 var v struct {Blocked bool `bson:"blocked"`}
 err:=c.db.Collection("BlockedUsers").FindOne(c.ctx,bson.M{"_id":id}).Decode(&v)
 if err!=nil {
  if errors.Is(err,mongo.ErrNoDocuments) {return false,nil}
  return false,err
 }
 return v.Blocked,nil
}
