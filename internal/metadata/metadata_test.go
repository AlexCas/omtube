package metadata

import (
	"testing"

	"github.com/alexcasdev/terminaltube/internal/search"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		name       string
		in         search.Result
		wantArtist string
		wantTitle  string
	}{
		{
			name:       "split simple artista y título",
			in:         search.Result{Title: "Artist - Song"},
			wantArtist: "Artist",
			wantTitle:  "Song",
		},
		{
			name:       "split con guion largo (en-dash)",
			in:         search.Result{Title: "Daft Punk – Get Lucky"},
			wantArtist: "Daft Punk",
			wantTitle:  "Get Lucky",
		},
		{
			name:       "split toma el primer separador",
			in:         search.Result{Title: "A - B - C"},
			wantArtist: "A",
			wantTitle:  "B - C",
		},
		{
			name:       "elimina etiqueta (Official Music Video)",
			in:         search.Result{Title: "Artist - Song (Official Music Video)"},
			wantArtist: "Artist",
			wantTitle:  "Song",
		},
		{
			name:       "elimina (Official Video) y descarta feat.",
			in:         search.Result{Title: "Artist - Song (Official Video) feat. Other"},
			wantArtist: "Artist",
			wantTitle:  "Song",
		},
		{
			name:       "elimina [MV]",
			in:         search.Result{Title: "Artist - Song [MV]"},
			wantArtist: "Artist",
			wantTitle:  "Song",
		},
		{
			name:       "elimina (Lyrics) y (Lyric Video)",
			in:         search.Result{Title: "Artist - Song (Lyrics) (Lyric Video)"},
			wantArtist: "Artist",
			wantTitle:  "Song",
		},
		{
			name:       "elimina (Audio) (Visualizer) (HD)",
			in:         search.Result{Title: "Artist - Song (Audio) (Visualizer) (HD)"},
			wantArtist: "Artist",
			wantTitle:  "Song",
		},
		{
			name:       "elimina etiqueta de año",
			in:         search.Result{Title: "Artist - Song (2021)"},
			wantArtist: "Artist",
			wantTitle:  "Song",
		},
		{
			name:       "descarta ft. sin paréntesis",
			in:         search.Result{Title: "Artist - Song ft. Someone"},
			wantArtist: "Artist",
			wantTitle:  "Song",
		},
		{
			name:       "descarta (feat. X) entre paréntesis",
			in:         search.Result{Title: "Artist - Song (feat. X)"},
			wantArtist: "Artist",
			wantTitle:  "Song",
		},
		{
			name:       "artista derivado de canal VEVO",
			in:         search.Result{Title: "Song Title (Official Music Video)", Uploader: "ArtistVEVO"},
			wantArtist: "Artist",
			wantTitle:  "Song Title",
		},
		{
			name:       "artista derivado de canal - Topic",
			in:         search.Result{Title: "Song Title", Uploader: "Artist - Topic"},
			wantArtist: "Artist",
			wantTitle:  "Song Title",
		},
		{
			name:       "artista derivado quitando Official",
			in:         search.Result{Title: "Some Song", Uploader: "Artist Official"},
			wantArtist: "Artist",
			wantTitle:  "Some Song",
		},
		{
			name:       "colapsa espacios repetidos y extremos",
			in:         search.Result{Title: "  Artist   -   Song   Name  "},
			wantArtist: "Artist",
			wantTitle:  "Song Name",
		},
		{
			name:       "sin separador ni uploader",
			in:         search.Result{Title: "Just A Title"},
			wantArtist: "",
			wantTitle:  "Just A Title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotArtist, gotTitle := Normalize(tt.in)
			if gotArtist != tt.wantArtist {
				t.Errorf("artist = %q, want %q", gotArtist, tt.wantArtist)
			}
			if gotTitle != tt.wantTitle {
				t.Errorf("title = %q, want %q", gotTitle, tt.wantTitle)
			}
		})
	}
}

// TestNormalizeNonMutating verifica que Normalize no muta la entrada
// (requisito "Query-Only, Non-Mutating").
func TestNormalizeNonMutating(t *testing.T) {
	in := search.Result{
		ID:       "abc123",
		Title:    "ArtistVEVO - Song (Official Music Video) feat. Other",
		Uploader: "ArtistVEVO",
		Duration: 200,
	}
	want := in // copia de los valores originales

	_, _ = Normalize(in)

	if in != want {
		t.Fatalf("Normalize mutó la entrada: got %+v, want %+v", in, want)
	}
}
