package core

import "testing"

func TestTamilSearchQuery(t *testing.T) {
 cases := []struct { input, title string; prefer bool }{
  {"The A Team Tamil", "The A Team", true},
  {"The A Team Tamil Dub", "The A Team", true},
  {"The A Team Tamil Dubbed", "The A Team", true},
  {"The A Team tamil audio", "The A Team", true},
  {"The A Team TAMIL AUDIO", "The A Team", true},
  {"The A Team", "The A Team", false},
  {"Tamil", "Tamil", false},
  {"Tamil Movie", "Tamil Movie", false},
  {"The A Team Tamil Sub", "The A Team Tamil Sub", false},
 }
 for _, c := range cases {
  query := parseMovieSearch(c.input)
  title, prefer := query.title, query.audio == "tam"
  if title != c.title || prefer != c.prefer {
   t.Errorf("%q: got (%q,%v), want (%q,%v)", c.input,title,prefer,c.title,c.prefer)
  }
 }
}
