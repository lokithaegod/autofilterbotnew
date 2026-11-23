// Package captiontags recognizes manually declared Malayalam subtitle suitability.
// Such declarations are NOT proof of actual subtitle synchronization.
package captiontags

import (
 "path/filepath"
 "regexp"
 "strings"
)

var space=regexp.MustCompile(`\s+`)
var negation=regexp.MustCompile(`(?i)(?:^|[^a-z0-9])(?:no|without|not[\s._-]*available)[\s._-]+malayalam[\s._-]*(?:sub|subs|subtitle|subtitles)(?:$|[^a-z0-9])`)
var marker=regexp.MustCompile(`(?i)(?:^|[^a-z0-9])malayalam[\s._-]*(?:sub|subs|subtitle|subtitles)(?:$|[^a-z0-9])`)

// IsTagged matches "Malayalam Sub" in a video's Telegram caption.
func IsTagged(caption string) bool {
 caption=space.ReplaceAllString(strings.TrimSpace(caption)," ")
 return marker.MatchString(caption) && !negation.MatchString(caption)
}

// IsVideoUpload excludes .srt and other non-video documents from manual tags.
func IsVideoUpload(fileName string, videoMedia bool) bool {
 if videoMedia{return true}
 switch strings.ToLower(filepath.Ext(fileName)){
 case ".mkv",".mp4",".avi",".mov",".webm",".m4v",".ts",".wmv":return true
 default:return false
 }
}
