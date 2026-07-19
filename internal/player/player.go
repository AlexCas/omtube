// Package player reproduce audio controlando un proceso mpv por IPC.
package player

import "github.com/alexcasdev/terminaltube/internal/search"

// EventKind clasifica los eventos emitidos por el reproductor.
type EventKind int

const (
	// EventEndFile indica que la pista actual terminó.
	EventEndFile EventKind = iota
	// EventLoaded indica que una nueva pista empezó a cargarse/reproducirse.
	EventLoaded
	// EventTrackChange indica que una nueva pista comenzó a cargarse; lleva el
	// metadato de la pista y la fuente cargada (archivo local o URL) para que los
	// suscriptores (letra, portada, presencia de Discord) reaccionen.
	EventTrackChange
)

// Event es una notificación asíncrona del reproductor hacia la UI. Para
// EventTrackChange, Track contiene el metadato de la pista y Source la fuente
// cargada (ruta de archivo local o URL de YouTube).
type Event struct {
	Kind   EventKind
	Track  search.Result
	Source string
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
	// Load carga y reproduce la fuente indicada (ruta de archivo local o URL/ID
	// de YouTube). Emite EventLoaded.
	Load(src string) error
	// LoadTrack carga la fuente src (archivo local cacheado o URL de YouTube) y
	// emite EventTrackChange con el metadato track, para que los suscriptores
	// (letra/portada/presencia) reaccionen al cambio de pista.
	LoadTrack(src string, track search.Result) error
	// TogglePause alterna entre pausa y reproducción.
	TogglePause() error
	// Stop detiene la reproducción actual dejando el proceso vivo y ocioso (a
	// diferencia de Close, que termina mpv). Usado al limpiar la cola.
	Stop() error
	// AddVolume ajusta el volumen en delta (clamp 0–130) y devuelve el nuevo valor.
	AddVolume(delta int) (int, error)
	// Seek salta el offset dado en segundos (relativo a la posición actual).
	Seek(offset float64) error
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
