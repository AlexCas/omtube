package artwork

import (
	"context"
	"testing"
)

// fakeEnv construye un lector de entorno a partir de un mapa, devolviendo cadena
// vacía para claves ausentes (como os.Getenv).
func fakeEnv(m map[string]string) env {
	return func(k string) string { return m[k] }
}

func TestDetectMatrix(t *testing.T) {
	// Detect nunca selecciona Kitty/Sixel porque su Render todavía no emite el
	// protocolo nativo (sería un placeholder): usa chafa cuando está disponible y
	// degrada a None en caso contrario, sin importar el terminal anunciado. Esto
	// mantiene la detección honesta con lo que Render realmente puede dibujar.
	cases := []struct {
		name     string
		env      map[string]string
		hasChafa bool
		want     Backend
	}{
		{"kitty env but chafa available ⇒ chafa", map[string]string{"KITTY_WINDOW_ID": "1"}, true, Chafa},
		{"kitty env without chafa ⇒ none", map[string]string{"KITTY_WINDOW_ID": "1"}, false, None},
		{"sixel env but chafa available ⇒ chafa", map[string]string{"TERM": "foot"}, true, Chafa},
		{"sixel env without chafa ⇒ none", map[string]string{"TERM": "xterm-sixel"}, false, None},
		{"chafa available", map[string]string{"TERM": "xterm-256color"}, true, Chafa},
		{"none", map[string]string{"TERM": "xterm-256color"}, false, None},
		{"empty env, chafa ⇒ chafa", map[string]string{}, true, Chafa},
		{"empty env, no chafa ⇒ none", map[string]string{}, false, None},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := detect(fakeEnv(c.env), func() bool { return c.hasChafa })
			if got != c.want {
				t.Fatalf("detect = %v, want %v", got, c.want)
			}
		})
	}
}

func TestDetectNeverAdvertisesUnrenderableBackend(t *testing.T) {
	// Ningún entorno debe hacer que Detect devuelva un backend cuyo Render sea un
	// placeholder (Kitty/Sixel). Con o sin chafa, solo Chafa o None son válidos.
	envs := []map[string]string{
		{"KITTY_WINDOW_ID": "1", "TERM": "xterm-kitty"},
		{"TERM_PROGRAM": "ghostty"},
		{"TERM": "foot", "TERM_FEATURES": "sixel"},
	}
	for _, e := range envs {
		for _, hasChafa := range []bool{true, false} {
			got := detect(fakeEnv(e), func() bool { return hasChafa })
			if got == Kitty || got == Sixel {
				t.Fatalf("detect(%v, chafa=%v) = %v; no debe anunciar Kitty/Sixel", e, hasChafa, got)
			}
		}
	}
}

func TestBackendString(t *testing.T) {
	cases := map[Backend]string{Kitty: "kitty", Sixel: "sixel", Chafa: "chafa", None: "none"}
	for b, want := range cases {
		if got := b.String(); got != want {
			t.Errorf("%d.String() = %q, want %q", b, got, want)
		}
	}
}

func TestRenderEmptyURL(t *testing.T) {
	for _, b := range []Backend{Kitty, Sixel, Chafa, None} {
		out, err := b.Render(context.Background(), "", 20, 10)
		if err != nil {
			t.Fatalf("Render(%v) err = %v, want nil", b, err)
		}
		if out != placeholder {
			t.Fatalf("Render(%v) con URL vacía = %q, want placeholder", b, out)
		}
	}
}

func TestRenderNoneDegrades(t *testing.T) {
	out, err := None.Render(context.Background(), "https://example.com/cover.jpg", 20, 10)
	if err != nil {
		t.Fatalf("None.Render err = %v, want nil", err)
	}
	if out != placeholder {
		t.Fatalf("None.Render = %q, want placeholder", out)
	}
}

func TestRenderChafaUnavailableDegrades(t *testing.T) {
	// renderChafa nunca debe propagar error aunque el binario o la imagen
	// fallen; con una URL inexistente devuelve el placeholder.
	out, err := renderChafa(context.Background(), "http://127.0.0.1:0/nope.jpg", 10, 5)
	if err != nil {
		t.Fatalf("renderChafa err = %v, want nil", err)
	}
	if out == "" {
		t.Fatalf("renderChafa devolvió cadena vacía, se esperaba contenido o placeholder")
	}
}

func TestSizeArgDefaults(t *testing.T) {
	if got := sizeArg(0, 0); got != "20x10" {
		t.Fatalf("sizeArg(0,0) = %q, want 20x10", got)
	}
	if got := sizeArg(40, 24); got != "40x24" {
		t.Fatalf("sizeArg(40,24) = %q, want 40x24", got)
	}
}
