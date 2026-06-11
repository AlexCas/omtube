package search

import "testing"

func TestParseEntries(t *testing.T) {
	data := []byte(`{"id":"abc","title":"Numb","uploader":"Linkin Park","duration":185.0}
{"id":"def","title":"In The End","channel":"LP Channel","duration":216}

{"title":"sin id","duration":100}
{malformed json}
{"id":"ghi","title":"Faint","duration":162}`)

	res, err := parseEntries(data)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if len(res) != 3 {
		t.Fatalf("se esperaban 3 resultados, got %d: %+v", len(res), res)
	}
	if res[0].ID != "abc" || res[0].Title != "Numb" || res[0].Uploader != "Linkin Park" || res[0].Duration != 185 {
		t.Fatalf("primer resultado mal parseado: %+v", res[0])
	}
	// uploader vacío usa channel como fallback.
	if res[1].Uploader != "LP Channel" {
		t.Fatalf("fallback a channel falló: %+v", res[1])
	}
}

func TestResultURL(t *testing.T) {
	r := Result{ID: "xyz"}
	if got := r.URL(); got != "https://www.youtube.com/watch?v=xyz" {
		t.Fatalf("URL = %s", got)
	}
}
