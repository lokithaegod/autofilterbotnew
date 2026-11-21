package autofilter

import (
 "regexp"
 "sort"
 "strconv"
 "strings"

 "github.com/Jisin0/autofilterbot/internal/model"
)

// Recognizes common episode filenames such as Show.S01E02 and Show.1x02.
// A leading show title is used so unrelated series do not get mixed.
var seasonEpisode = regexp.MustCompile(`(?i)(?:^|[^a-z0-9])s([0-9]{1,2})[ ._-]*e([0-9]{1,3})(?:[^a-z0-9]|$)`)
var xEpisode = regexp.MustCompile(`(?i)(?:^|[^a-z0-9])([0-9]{1,2})x([0-9]{1,3})(?:[^a-z0-9]|$)`)
var titleDivider = regexp.MustCompile(`[^a-z0-9]+`)

type episodeKey struct{
 title string
 season int
 episode int
 valid bool
}

func parseEpisode(name string) episodeKey {
 lower:=strings.ToLower(name)
 loc:=seasonEpisode.FindStringSubmatchIndex(lower)
 if loc==nil {loc=xEpisode.FindStringSubmatchIndex(lower)}
 if loc==nil {return episodeKey{}}
 match:=lower[loc[0]:loc[1]]
 nums:=regexp.MustCompile(`[0-9]+`).FindAllString(match,-1)
 if len(nums)<2 {return episodeKey{}}
 season,err1:=strconv.Atoi(nums[0]);episode,err2:=strconv.Atoi(nums[1])
 if err1!=nil||err2!=nil||episode==0 {return episodeKey{}}
 title:=strings.TrimSpace(titleDivider.ReplaceAllString(lower[:loc[0]]," "))
 if title=="" {return episodeKey{}}
 return episodeKey{title:title,season:season,episode:episode,valid:true}
}

// EpisodeLess sorts matching TV files by title, season and episode number.
// For non-episode files, it maintains the previous relative position.
func EpisodeLess(a,b string) bool {
 x,y:=parseEpisode(a),parseEpisode(b)
 if x.valid && y.valid {
  if x.title!=y.title {return x.title<y.title}
  if x.season!=y.season {return x.season<y.season}
  if x.episode!=y.episode {return x.episode<y.episode}
 }
 return false
}

// SortEpisodes orders a result set only when it contains a recognized episode.
// Other search result sets preserve their original order.
func SortEpisodes(pages []Files) {
 var flat []File
 var sizes []int
 episodes:=0
 for _,p:=range pages {
  sizes=append(sizes,len(p))
  for _,f:=range p {
   flat=append(flat,f)
   if parseEpisode(f.FileName).valid {episodes++}
  }
 }
 if episodes==0{return}
 // Keep unrecognized filenames after episode files, preserving their order.
 sort.SliceStable(flat,func(i,j int)bool{
  a,b:=parseEpisode(flat[i].FileName),parseEpisode(flat[j].FileName)
  if a.valid!=b.valid{return a.valid}
  if a.valid&&b.valid{return EpisodeLess(flat[i].FileName,flat[j].FileName)}
  return false
 })
 offset:=0
 for i,n:=range sizes {
  copy(pages[i],flat[offset:offset+n])
  offset+=n
 }
}

// SortModelEpisodes is used by the /find and /browse catalog.
func SortModelEpisodes(files []model.File) {
 sort.SliceStable(files,func(i,j int)bool{
  a,b:=parseEpisode(files[i].FileName),parseEpisode(files[j].FileName)
  if a.valid!=b.valid{return a.valid}
  if a.valid&&b.valid{return EpisodeLess(files[i].FileName,files[j].FileName)}
  return false
 })
}
