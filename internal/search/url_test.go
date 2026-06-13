package search

import "testing"

func TestClassifyURL(t *testing.T) {
	cases := []struct {
		name     string
		raw      string
		wantKind URLKind
		wantID   string
	}{
		{"watch", "https://www.youtube.com/watch?v=abc123", URLVideo, "abc123"},
		{"watch no www", "https://youtube.com/watch?v=abc123", URLVideo, "abc123"},
		{"watch with list is video", "https://www.youtube.com/watch?v=abc123&list=PL999", URLVideo, "abc123"},
		{"youtu.be", "https://youtu.be/abc123", URLVideo, "abc123"},
		{"youtu.be with query", "https://youtu.be/abc123?t=42", URLVideo, "abc123"},
		{"shorts", "https://www.youtube.com/shorts/abc123", URLVideo, "abc123"},
		{"embed", "https://www.youtube.com/embed/abc123", URLVideo, "abc123"},
		{"music host watch", "https://music.youtube.com/watch?v=abc123", URLVideo, "abc123"},
		{"playlist", "https://www.youtube.com/playlist?list=PL999", URLPlaylist, "PL999"},
		{"watch without v but list", "https://www.youtube.com/watch?list=PL999", URLPlaylist, "PL999"},
		{"non youtube", "https://example.com/watch?v=abc123", URLUnknown, ""},
		{"empty", "", URLUnknown, ""},
		{"garbage", "not a url at all", URLUnknown, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			kind, id := ClassifyURL(tc.raw)
			if kind != tc.wantKind || id != tc.wantID {
				t.Fatalf("ClassifyURL(%q) = (%v, %q); want (%v, %q)", tc.raw, kind, id, tc.wantKind, tc.wantID)
			}
		})
	}
}
