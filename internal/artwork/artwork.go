// Package artwork detecta la capacidad gráfica del terminal (kitty graphics,
// sixel o chafa) y renderiza la portada de la pista, degradando con elegancia a
// un placeholder cuando no hay soporte. Las funciones de detección y render son
// puras y unit-testeables: nunca devuelven error que interrumpa la reproducción.
package artwork

import (
	"context"
	"os"
	"os/exec"
	"strconv"
)

// Backend identifica el protocolo gráfico seleccionado para el terminal actual.
type Backend int

const (
	// None: el terminal no soporta imágenes ni hay chafa disponible.
	None Backend = iota
	// Kitty: protocolo de gráficos de kitty.
	Kitty
	// Sixel: gráficos sixel.
	Sixel
	// Chafa: degradación a bloques/ASCII mediante el binario chafa.
	Chafa
)

// String da un nombre legible al backend (útil en logs y tests).
func (b Backend) String() string {
	switch b {
	case Kitty:
		return "kitty"
	case Sixel:
		return "sixel"
	case Chafa:
		return "chafa"
	default:
		return "none"
	}
}

// env abstrae la lectura del entorno para poder inyectar un matriz de variables
// en los tests sin tocar el proceso real.
type env func(string) string

// Detect inspecciona el entorno del proceso y elige el backend de render.
func Detect() Backend { return detect(os.Getenv, chafaAvailable) }

// detect es la forma testeable de Detect: recibe el lector de entorno y un
// predicado que indica si chafa está disponible.
//
// Aunque el enum conserva los backends Kitty y Sixel, Detect NO los selecciona:
// su Render todavía no emite las secuencias de escape del protocolo nativo
// (kitty graphics / sixel) y solo devolvería un placeholder. Anunciar esa
// capacidad sin poder cumplirla sería deshonesto, así que la detección usa chafa
// cuando el binario está disponible y degrada a None en caso contrario. El
// soporte nativo de kitty/sixel queda como mejora futura. El argumento getenv se
// mantiene por compatibilidad y para una futura reactivación de la detección por
// terminal.
func detect(getenv env, hasChafa func() bool) Backend {
	_ = getenv
	if hasChafa() {
		return Chafa
	}
	return None
}

// chafaAvailable indica si el binario chafa está en el PATH.
func chafaAvailable() bool {
	_, err := exec.LookPath("chafa")
	return err == nil
}

// placeholder es el contenido devuelto cuando no se puede renderizar una
// imagen; la UI lo muestra en el panel de portada.
const placeholder = "[sin portada]"

// Render produce la representación de la portada en thumbURL para un área de
// w×h celdas según el backend del terminal. Para None o ante cualquier fallo
// (URL vacía, chafa ausente, error de proceso) devuelve un placeholder y
// err=nil: la portada nunca interrumpe la reproducción.
func (b Backend) Render(ctx context.Context, thumbURL string, w, h int) (string, error) {
	if thumbURL == "" {
		return placeholder, nil
	}
	switch b {
	case Chafa:
		return renderChafa(ctx, thumbURL, w, h)
	case Kitty, Sixel:
		// El protocolo gráfico requiere bytes de la imagen ya descargada y la
		// orquestación de escape sequences contra el terminal real, que vive en
		// la capa de UI (Fase 3). Aquí se devuelve un placeholder estable para
		// mantener la función pura y testeable sin un terminal real.
		return placeholder, nil
	default:
		return placeholder, nil
	}
}

// renderChafa invoca chafa para convertir la imagen en bloques/ASCII del tamaño
// pedido. Cualquier error (binario ausente, descarga fallida) degrada al
// placeholder sin propagar el error.
func renderChafa(ctx context.Context, thumbURL string, w, h int) (string, error) {
	bin, err := exec.LookPath("chafa")
	if err != nil {
		return placeholder, nil
	}
	args := []string{"--format=symbols", "--size", sizeArg(w, h), thumbURL}
	out, err := exec.CommandContext(ctx, bin, args...).Output()
	if err != nil || len(out) == 0 {
		return placeholder, nil
	}
	return string(out), nil
}

// sizeArg formatea el argumento WxH para chafa, usando valores por defecto
// razonables cuando las dimensiones no son positivas.
func sizeArg(w, h int) string {
	if w <= 0 {
		w = 20
	}
	if h <= 0 {
		h = 10
	}
	return strconv.Itoa(w) + "x" + strconv.Itoa(h)
}
