// Package player reproduce audio controlando un proceso mpv por IPC.
package player

// EventKind clasifica los eventos emitidos por el reproductor.
type EventKind int

const (
	// EventEndFile indica que la pista actual terminó.
	EventEndFile EventKind = iota
	// EventLoaded indica que una nueva pista empezó a cargarse/reproducirse.
	EventLoaded
)

// Event es una notificación asíncrona del reproductor hacia la UI.
type Event struct {
	Kind EventKind
}

// State es una instantánea del estado de reproducción.
type State struct {
	Paused bool
	Volume int
	Pos    float64 // segundos transcurridos
	Dur    float64 // duración total en segundos
}

// Player controla la reproducción de audio.
type Player interface {
	// Load carga y reproduce la pista identificada por su URL/ID de YouTube.
	Load(url string) error
	// TogglePause alterna entre pausa y reproducción.
	TogglePause() error
	// AddVolume ajusta el volumen en delta (clamp 0–130) y devuelve el nuevo valor.
	AddVolume(delta int) (int, error)
	// Position devuelve la posición y duración actuales en segundos.
	Position() (pos, dur float64)
	// Paused indica si la reproducción está pausada.
	Paused() bool
	// Volume devuelve el volumen actual.
	Volume() int
	// Events expone el canal de eventos del reproductor.
	Events() <-chan Event
	// Close detiene mpv y libera recursos.
	Close() error
}
