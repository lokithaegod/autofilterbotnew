package captiontags

import (
	"regexp"
	"strings"
)

// Tamil dubbed refers to a dubbed VIDEO, not a Tamil-language subtitle.
// Explicit negative captions must never create a Tamil dubbed label.
var tamilDubCaptionNegative = regexp.MustCompile(`(?i)(?:^|[^a-z0-9])(?:no|without|not[\s._-]*available)[\s._-]*(?:tamil|tam)[\s._-]*(?:dub|dubbed|dubbing)(?:$|[^a-z0-9])`)

var tamilDubCaptionTrailingNegative = regexp.MustCompile(`(?i)(?:tamil|tam)[\s._-]*(?:dub|dubbed|dubbing)[\s._-]*(?:not[\s._-]*available|unavailable|no)(?:$|[^a-z0-9])`)

// IsTamilDubbedCaption recognizes affirmative Tamil audio and dub labels
// for the existing Tamil section.
func IsTamilDubbedCaption(caption string) bool {
	cleaned := space.ReplaceAllString(strings.TrimSpace(caption), " ")
	if tamilDubCaptionNegative.MatchString(cleaned) || tamilDubCaptionTrailingNegative.MatchString(cleaned) {
		return false
	}
	return AudioLanguages("", caption)["tam"]
}
