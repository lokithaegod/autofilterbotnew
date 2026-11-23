// Package compatfinder performs read-only, evidence-gated movie/subtitle candidate matching.
package compatfinder

import (
 "regexp"
 "strings"
 "path/filepath"
)
var titleYear=regexp.MustCompile(`(?i)^(.+?)[ ._\-(\[]+((?:19|20)[0-9]{2})(?:[^0-9]|$)`)
var cleanup=regexp.MustCompile(`[^a-z0-9]+`)
func Normalize(value string)string{return strings.Join(strings.Fields(cleanup.ReplaceAllString(strings.ToLower(value)," "))," ")}
func Identity(filename string)(title,year string,ok bool){
 stem:=strings.TrimSuffix(filename,filepath.Ext(filename))
 m:=titleYear.FindStringSubmatch(stem)
 if len(m)!=3{return "","",false}
 return Normalize(m[1]),m[2],Normalize(m[1])!=""
}
type Edition string
const(EditionUnknown Edition=""; EditionIMAX Edition="imax";EditionExtended Edition="extended";EditionDirectors Edition="directors-cut";EditionTheatrical Edition="theatrical")
func DetectEdition(name string) Edition {
 key:=Normalize(name)
 switch{
 case strings.Contains(key,"imax"):return EditionIMAX
 case strings.Contains(key,"extended")||strings.Contains(key,"uncut"):return EditionExtended
 case strings.Contains(key,"directors cut")||strings.Contains(key,"director cut"):return EditionDirectors
 case strings.Contains(key,"theatrical"):return EditionTheatrical
 }
 return EditionUnknown
}
type Candidate struct {FileID,UniqueID,Name,Kind string}
type Result struct {Video Candidate; Status,Reason,Confidence string; Offset,Drift float64}
func Classify(subName,videoName string)(status,reason string){
 st,sy,sok:=Identity(subName)
 vt,vy,vok:=Identity(videoName)
 if !sok||!vok{return "unverified","Missing title or year"}
 if st!=vt||sy!=vy{return "incompatible","Different title or release year"}
 se,ve:=DetectEdition(subName),DetectEdition(videoName)
 if se!=""&&ve!=""&&se!=ve {return "incompatible","Explicitly different movie editions"}
 return "unverified","Title and year match; audio timing has not been checked"
}
