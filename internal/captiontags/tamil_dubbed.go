package captiontags

import (
 "regexp"
 "strings"
)

// Tamil dubbed refers to a dubbed VIDEO, not a Tamil-language subtitle.
// Explicit negative captions must never create a Tamil dubbed label.
var tamilDubCaptionMarker=regexp.MustCompile(`(?i)(?:^|[^a-z0-9])(?:tamil|tam)[\s._-]*(?:dub|dubbed|dubbing)(?:$|[^a-z0-9])`)
var tamilDubCaptionNegative=regexp.MustCompile(`(?i)(?:^|[^a-z0-9])(?:no|without|not[\s._-]*available)[\s._-]*(?:tamil|tam)[\s._-]*(?:dub|dubbed|dubbing)(?:$|[^a-z0-9])`)

var tamilDubCaptionTrailingNegative=regexp.MustCompile(`(?i)(?:tamil|tam)[\s._-]*(?:dub|dubbed|dubbing)[\s._-]*(?:not[\s._-]*available|unavailable|no)(?:$|[^a-z0-9])`)

var tamilAudioWord=regexp.MustCompile(`(?i)\btamil\b`)
var tamilNonAudio=regexp.MustCompile(`(?i)\b(?:(?:no|without)\s+tamil(?:\s+audio)?|tamil\s+(?:subtitles?|subs?|audio\s+(?:unavailable|not available)|not available|unavailable))\b`)

// IsTamilDubbedCaption detects explicit affirmative Tamil-dubbed claims.
func IsTamilDubbedCaption(caption string)bool{
 cleaned:=space.ReplaceAllString(strings.TrimSpace(caption)," ")
 if tamilDubCaptionNegative.MatchString(cleaned)||tamilDubCaptionTrailingNegative.MatchString(cleaned){return false}
 if tamilDubCaptionMarker.MatchString(cleaned)||strings.Contains(cleaned,"தமிழ் டப்"){return true}
 return tamilAudioWord.MatchString(tamilNonAudio.ReplaceAllString(cleaned," "))
}
