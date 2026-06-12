package lyrics

import "testing"

func TestParseLRC(t *testing.T) {
	body := "[ar:Some Artist]\n[ti:Some Song]\n" +
		"[00:12.00]Primera línea\n" +
		"[00:17.20]Segunda línea\n" +
		"[01:05.50]Tercera línea\n"

	l := parseLRC(body)
	if !l.Synced {
		t.Fatalf("se esperaba letra sincronizada")
	}
	if len(l.Lines) != 3 {
		t.Fatalf("se esperaban 3 líneas, got %d: %+v", len(l.Lines), l.Lines)
	}
	want := []struct {
		t    float64
		text string
	}{
		{12.0, "Primera línea"},
		{17.2, "Segunda línea"},
		{65.5, "Tercera línea"},
	}
	for i, w := range want {
		if l.Lines[i].T != w.t || l.Lines[i].Text != w.text {
			t.Errorf("línea %d = {%v %q}, want {%v %q}", i, l.Lines[i].T, l.Lines[i].Text, w.t, w.text)
		}
	}
}

func TestParseLRCMultiStampAndOrdering(t *testing.T) {
	// Marcas múltiples por línea y fuera de orden: deben expandirse y ordenarse.
	body := "[00:30.00]Estribillo\n[00:10.00][00:50.00]Estribillo repetido\n"
	l := parseLRC(body)
	if len(l.Lines) != 3 {
		t.Fatalf("se esperaban 3 líneas tras expandir, got %d", len(l.Lines))
	}
	prev := -1.0
	for _, ln := range l.Lines {
		if ln.T < prev {
			t.Fatalf("líneas no ordenadas por tiempo: %+v", l.Lines)
		}
		prev = ln.T
	}
	if l.Lines[0].T != 10.0 || l.Lines[1].T != 30.0 || l.Lines[2].T != 50.0 {
		t.Fatalf("orden temporal incorrecto: %+v", l.Lines)
	}
}

func TestParseLRCNoTimestamps(t *testing.T) {
	// Solo metadatos: no es sincronizado.
	l := parseLRC("[ar:X]\n[ti:Y]\nNo es lrc")
	if l.Synced || !l.Empty() {
		t.Fatalf("se esperaba Lyrics vacía/no sincronizada, got %+v", l)
	}
}

func TestPlainText(t *testing.T) {
	if l := plainText("  \n  "); !l.Empty() {
		t.Fatalf("texto en blanco debe ser Lyrics vacía, got %+v", l)
	}
	l := plainText("Línea 1\nLínea 2")
	if l.Synced || l.Empty() || l.Plain != "Línea 1\nLínea 2" {
		t.Fatalf("plainText incorrecto: %+v", l)
	}
}

func TestLineAt(t *testing.T) {
	l := parseLRC("[00:10.00]A\n[00:20.00]B\n[00:30.00]C\n")

	cases := []struct {
		sec  float64
		want int
	}{
		{0, -1},     // antes de la primera marca
		{9.9, -1},   // justo antes de la primera
		{10, 0},     // exactamente en la primera
		{15, 0},     // entre la primera y la segunda
		{20, 1},     // exactamente en la segunda
		{29.999, 1}, // justo antes de la tercera
		{30, 2},     // exactamente en la tercera
		{120, 2},    // muy después: se mantiene en la última
	}
	for _, c := range cases {
		if got := l.LineAt(c.sec); got != c.want {
			t.Errorf("LineAt(%v) = %d, want %d", c.sec, got, c.want)
		}
	}
}

func TestLineAtSeekIsStable(t *testing.T) {
	// Un salto hacia atrás debe resolver la línea correcta igual que el avance.
	l := parseLRC("[00:05.00]A\n[00:15.00]B\n[00:25.00]C\n")
	if got := l.LineAt(26); got != 2 {
		t.Fatalf("avance: LineAt(26) = %d, want 2", got)
	}
	if got := l.LineAt(6); got != 0 {
		t.Fatalf("seek atrás: LineAt(6) = %d, want 0", got)
	}
}

func TestLineAtUnsynced(t *testing.T) {
	if got := plainText("solo texto").LineAt(10); got != -1 {
		t.Fatalf("LineAt en letra no sincronizada = %d, want -1", got)
	}
	if got := (Lyrics{}).LineAt(10); got != -1 {
		t.Fatalf("LineAt en Lyrics vacía = %d, want -1", got)
	}
}

func TestParseTimestampVariants(t *testing.T) {
	cases := []struct {
		in   string
		want float64
		ok   bool
	}{
		{"00:12.00", 12, true},
		{"01:05.50", 65.5, true},
		{"02:00", 120, true},
		{"00:12:34", 12.34, true}, // mm:ss:cc
		{"ar:Artist", 0, false},
		{"ti:Title", 0, false},
		{"nope", 0, false},
	}
	for _, c := range cases {
		got, ok := parseTimestamp(c.in)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("parseTimestamp(%q) = (%v,%v), want (%v,%v)", c.in, got, ok, c.want, c.ok)
		}
	}
}
