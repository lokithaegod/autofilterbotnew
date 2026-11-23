// Package subtitles matches Malayalam subtitle uploads to movie release filenames.
// Filename comparisons indicate likelihood only, never verified synchronization.
package subtitles

import (
    "path/filepath"
    "regexp"
    "sort"
    "strings"
)

var yearPattern = regexp.MustCompile(`(?i)^(.+?)[ ._\-(\[]+((?:19|20)[0-9]{2})(?:[^0-9]|$)`)
var words = regexp.MustCompile(`[^a-z0-9]+`)
var fpsPattern = regexp.MustCompile(`(?i)(?:^|[^0-9])((?:23[.]976|24[.]000|25[.]000|29[.]970|30[.]000|23|24|25|30|60))\s*fps(?:[^a-z]|$)`)
var groupPattern = regexp.MustCompile(`(?i)(?:^|[^a-z0-9])\[?yts(?:[.]ag|[.]mx)?\]?(?:[^a-z0-9]|$)`)
var trailingGroup = regexp.MustCompile(`(?i)-([a-z][a-z0-9]{1,19})$`)

type Entry struct {
    UniqueID string `bson:"_id"`
    FileID string `bson:"file_id"`
    FileName string `bson:"file_name"`
    Title string `bson:"title"`
    TitleKey string `bson:"title_key"`
    Year string `bson:"year"`
    Size int64 `bson:"file_size"`
    ChatID int64 `bson:"chat_id"`
    MessageID int64 `bson:"message_id"`
    ReleaseType string `bson:"release_type,omitempty"`
    Resolution string `bson:"resolution,omitempty"`
    ReleaseGroup string `bson:"release_group,omitempty"`
    FPS string `bson:"fps,omitempty"`
}

func Supported(name string) bool {
    switch strings.ToLower(filepath.Ext(name)) {
    case ".srt", ".ass", ".vtt":
        return true
    }
    return false
}

func Normalize(s string) string {
    return strings.Join(strings.Fields(words.ReplaceAllString(strings.ToLower(s), " ")), " ")
}

// Parse requires the release year to prevent mixing similarly titled films.
func Parse(filename string) (title, year, key string, ok bool) {
    if !Supported(filename) {
        return "", "", "", false
    }
    stem := strings.TrimSuffix(filename, filepath.Ext(filename))
    matches := yearPattern.FindStringSubmatch(stem)
    if len(matches) != 3 {
        return "", "", "", false
    }
    raw := strings.TrimSpace(strings.Trim(matches[1], " ._-[]()"))
    title = strings.Join(strings.Fields(strings.NewReplacer(".", " ", "_", " ").Replace(raw)), " ")
    key = Normalize(title)
    if key == "" {
        return "", "", "", false
    }
    return title, matches[2], key, true
}

func SearchTerms(query string) (key, year string) {
    query = strings.TrimSpace(query)
    if m := yearPattern.FindStringSubmatch(query); len(m) == 3 {
        return Normalize(m[1]), m[2]
    }
    return Normalize(query), ""
}

type Release struct {
    Type, Resolution, Group, FPS string
}

type Candidate struct {
    Entry *Entry
    Level string
    Score int
    Reason string
}

func ReleaseFromFilename(filename string) Release {
    stem := strings.TrimSuffix(filename, filepath.Ext(filename))
    normalized := Normalize(stem)
    tokens := strings.Fields(normalized)
    has := func(value string) bool {
        for _, token := range tokens {
            if token == value {
                return true
            }
        }
        return false
    }
    var out Release
    switch {
    case strings.Contains(normalized, "web dl") || has("webdl") || strings.Contains(normalized, "web download"):
        out.Type = "WEB-DL"
    case has("webrip") || strings.Contains(normalized, "web rip"):
        out.Type = "WEBRip"
    case has("brrip"):
        out.Type = "BRRip"
    case has("bdrip"):
        out.Type = "BDRip"
    case has("hdrip"):
        out.Type = "HDRip"
    case has("bdremux") || has("remux"):
        out.Type = "BDRemux"
    case has("bluray") || strings.Contains(normalized, "blu ray"):
        out.Type = "BluRay"
    case has("hdtv") || has("pdtv"):
        out.Type = "HDTV"
    case has("dvdrip") || has("dvd"):
        out.Type = "DVDRip"
    }
    for _, resolution := range []string{"2160p", "1080p", "720p", "480p"} {
        if has(resolution) {
            out.Resolution = resolution
            break
        }
    }
    if m := fpsPattern.FindStringSubmatch(strings.ToLower(stem)); len(m) == 2 {
        out.FPS = m[1]
    }
    if groupPattern.MatchString(" " + strings.ToLower(stem) + " ") {
        out.Group = "yts"
    } else if m := trailingGroup.FindStringSubmatch(stem); len(m) == 2 {
        group := strings.ToLower(m[1])
        switch group {
        case "malayalam", "sub", "subs", "subtitle", "srt", "ass", "vtt", "720p", "1080p", "2160p":
        default:
            out.Group = group
        }
    }
    // Uploader handles such as @Team_Goat_Bot are not release groups.
    return out
}

func sourceFamily(kind string) string {
    switch kind {
    case "BluRay", "BRRip", "BDRip", "BDRemux":
        return "disc"
    case "WEB-DL", "WEBRip":
        return "web"
    default:
        return kind
    }
}

// Compare checks matching source families, but BluRay and BRRip are only
// related sources, not necessarily the same cut/timing or exact release.
func Compare(video, sub Release) (level string, score int, reason string) {
    var matched, conflicts, caveats []string
    known := 0
    if video.Type != "" && sub.Type != "" {
        known++
        if video.Type == sub.Type {
            score += 4
            matched = append(matched, "source matches")
        } else if sourceFamily(video.Type) == sourceFamily(sub.Type) {
            score += 2
            caveats = append(caveats, "related source types, different encodes possible")
        } else {
            conflicts = append(conflicts, "source differs")
        }
    }
    if video.Group != "" && sub.Group != "" {
        known++
        if strings.EqualFold(video.Group, sub.Group) {
            score += 3
            matched = append(matched, "release group matches")
        } else {
            caveats = append(caveats, "release group differs")
        }
    }
    if video.FPS != "" && sub.FPS != "" {
        known++
        if video.FPS == sub.FPS {
            score += 3
            matched = append(matched, "FPS matches")
        } else {
            conflicts = append(conflicts, "FPS differs")
        }
    }
    if video.Resolution != "" && sub.Resolution != "" {
        known++
        if video.Resolution == sub.Resolution {
            score++
            matched = append(matched, "resolution matches")
        } else {
            caveats = append(caveats, "resolution differs (not proof of timing mismatch)")
        }
    }
    if len(conflicts) > 0 {
        return "different", score, strings.Join(conflicts, ", ")
    }
    details := strings.Join(append(matched, caveats...), ", ")
    if known == 0 {
        return "unknown", score, "Release details unavailable"
    }
    if len(caveats) == 0 && score >= 4 {
        return "likely", score, details
    }
    if details == "" {
        details = "Limited release evidence"
    }
    return "possible", score, details
}

func priority(level string) int {
    switch level {
    case "likely":
        return 3
    case "possible":
        return 2
    case "unknown":
        return 1
    }
    return 0
}

func Rank(entries []*Entry, videos []string) []Candidate {
    results := make([]Candidate, 0, len(entries))
    for _, entry := range entries {
        sub := ReleaseFromFilename(entry.FileName)
        if entry.ReleaseType != "" {
            sub.Type = entry.ReleaseType
        }
        if entry.Resolution != "" {
            sub.Resolution = entry.Resolution
        }
        if entry.ReleaseGroup != "" {
            sub.Group = entry.ReleaseGroup
        }
        if entry.FPS != "" {
            sub.FPS = entry.FPS
        }
        best := Candidate{Entry: entry, Level: "unknown", Reason: "No movie release information available"}
        for index, name := range videos {
            level, score, reason := Compare(ReleaseFromFilename(name), sub)
            next := Candidate{Entry: entry, Level: level, Score: score, Reason: reason}
            if index == 0 || priority(next.Level) > priority(best.Level) ||
                (priority(next.Level) == priority(best.Level) && next.Score > best.Score) {
                best = next
            }
        }
        results = append(results, best)
    }
    sort.SliceStable(results, func(i, j int) bool {
        if priority(results[i].Level) != priority(results[j].Level) {
            return priority(results[i].Level) > priority(results[j].Level)
        }
        if results[i].Score != results[j].Score {
            return results[i].Score > results[j].Score
        }
        return results[i].Entry.FileName < results[j].Entry.FileName
    })
    return results
}
