package mpris

import (
	"testing"

	"github.com/alexcasdev/terminaltube/internal/lyrics"
	"github.com/alexcasdev/terminaltube/internal/search"
)

func TestMetadataDict(t *testing.T) {
	track := search.Result{
		ID:       "abc123",
		Title:    "Song Title",
		Uploader: "Artist Name",
		Duration: 185,
	}
	syncedLyrics := lyrics.Lyrics{
		Synced: true,
		Lines: []lyrics.Line{
			{T: 0, Text: "first line"},
			{T: 10, Text: "second line"},
		},
	}

	cases := []struct {
		name     string
		track    search.Result
		lyrics   lyrics.Lyrics
		wantKeys []string
		wantText []string
	}{
		{
			name:     "basic metadata without lyrics",
			track:    track,
			wantKeys: []string{"xesam:title", "xesam:artist", "xesam:album", "mpris:length", "mpris:artUrl"},
		},
		{
			name:     "metadata with synced lyrics includes asText",
			track:    track,
			lyrics:   syncedLyrics,
			wantKeys: []string{"xesam:title", "xesam:artist", "xesam:album", "mpris:length", "mpris:artUrl", "xesam:asText"},
			wantText: []string{"first line", "second line"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := metadataDict(tc.track, tc.lyrics)
			for _, k := range tc.wantKeys {
				if _, ok := d[k]; !ok {
					t.Errorf("missing key %q", k)
				}
			}
			if got := d["xesam:title"].Value(); got != track.Title {
				t.Errorf("title = %v, want %v", got, track.Title)
			}
			if got := d["xesam:artist"].Value(); got != track.Uploader {
				t.Errorf("artist = %v, want %v", got, track.Uploader)
			}
			if got := d["mpris:length"].Value(); got != int64(185)*1e6 {
				t.Errorf("length = %v, want %v", got, int64(185)*1e6)
			}
			wantURL := "https://i.ytimg.com/vi/abc123/hqdefault.jpg"
			if got := d["mpris:artUrl"].Value(); got != wantURL {
				t.Errorf("artUrl = %v, want %v", got, wantURL)
			}
			if tc.wantText != nil {
				text, ok := d["xesam:asText"].Value().([]string)
				if !ok {
					t.Fatalf("asText type = %T, want []string", d["xesam:asText"].Value())
				}
				if len(text) != len(tc.wantText) {
					t.Fatalf("asText len = %d, want %d", len(text), len(tc.wantText))
				}
				for i, want := range tc.wantText {
					if text[i] != want {
						t.Errorf("asText[%d] = %q, want %q", i, text[i], want)
					}
				}
			}
		})
	}
}

func TestMessageDispatch(t *testing.T) {
	s := newServer(func(interface{}) {}, nil)

	cases := []struct {
		name    string
		method  func()
		wantMsg interface{}
	}{
		{"PlayPause", func() { player{s}.PlayPause() }, PlayPauseMsg{}},
		{"Next", func() { player{s}.Next() }, NextMsg{}},
		{"Previous", func() { player{s}.Previous() }, PrevMsg{}},
		{"Stop", func() { player{s}.Stop() }, StopMsg{}},
		{"Pause", func() { player{s}.Pause() }, PlayPauseMsg{}},
		{"Play", func() { player{s}.Play() }, PlayPauseMsg{}},
		{"Seek", func() { player{s}.SeekRelative(30_000_000) }, SeekMsg{Offset: 30_000_000}},
		{"SetVolume", func() { player{s}.SetVolume(0.8) }, SetVolumeMsg{Volume: 0.8}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got interface{}
			s.send = func(msg interface{}) { got = msg }
			tc.method()
			if !messagesEqual(got, tc.wantMsg) {
				t.Errorf("sent msg = %#v, want %#v", got, tc.wantMsg)
			}
		})
	}
}

func messagesEqual(a, b interface{}) bool {
	switch am := a.(type) {
	case PlayPauseMsg:
		_, ok := b.(PlayPauseMsg)
		return ok
	case NextMsg:
		_, ok := b.(NextMsg)
		return ok
	case PrevMsg:
		_, ok := b.(PrevMsg)
		return ok
	case StopMsg:
		_, ok := b.(StopMsg)
		return ok
	case SeekMsg:
		bm, ok := b.(SeekMsg)
		return ok && am.Offset == bm.Offset
	case SetVolumeMsg:
		bm, ok := b.(SetVolumeMsg)
		return ok && am.Volume == bm.Volume
	}
	return false
}

func TestVolumeRoundTrip(t *testing.T) {
	cases := []struct {
		name      string
		playerVol int
		mprisVol  float64
	}{
		{"zero", 0, 0.0},
		{"half", 65, 0.5},
		{"max", 130, 1.0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotMpris := float64(tc.playerVol) / 130.0
			if gotMpris != tc.mprisVol {
				t.Errorf("player->mpris: got %v, want %v", gotMpris, tc.mprisVol)
			}
			gotPlayer := int(tc.mprisVol * 130)
			if gotPlayer != tc.playerVol {
				t.Errorf("mpris->player: got %d, want %d", gotPlayer, tc.playerVol)
			}
		})
	}
}

func TestNilServer(t *testing.T) {
	var s *Server
	track := search.Result{ID: "x", Title: "T"}
	ly := lyrics.Lyrics{Synced: true, Lines: []lyrics.Line{{Text: "l"}}}
	s.SetMetadata(track, ly)
	s.SetPlaybackStatus("Playing")
	s.SetVolume(50)
	s.SetPosition(10)
	s.Seeked(1_000_000)
	s.Close()
}
