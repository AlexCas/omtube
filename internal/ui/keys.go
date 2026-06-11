package ui

import "github.com/charmbracelet/bubbles/key"

// keyMap define los atajos de la aplicación.
type keyMap struct {
	Search  key.Binding
	Enqueue key.Binding
	Toggle  key.Binding
	Next    key.Binding
	Prev    key.Binding
	VolUp   key.Binding
	VolDown key.Binding
	Up      key.Binding
	Down    key.Binding
	Quit    key.Binding
	Cancel  key.Binding

	// Biblioteca (Fase 2).
	Library        key.Binding
	Favorite       key.Binding
	AddToPlaylist  key.Binding
	CreatePlaylist key.Binding
}

func defaultKeys() keyMap {
	return keyMap{
		Search:  key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "buscar")),
		Enqueue: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "encolar")),
		Toggle:  key.NewBinding(key.WithKeys(" "), key.WithHelp("espacio", "play/pausa")),
		Next:    key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "siguiente")),
		Prev:    key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "anterior")),
		VolUp:   key.NewBinding(key.WithKeys("+", "="), key.WithHelp("+", "vol+")),
		VolDown: key.NewBinding(key.WithKeys("-"), key.WithHelp("-", "vol-")),
		Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "arriba")),
		Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "abajo")),
		Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "salir")),
		Cancel:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancelar")),

		Library:        key.NewBinding(key.WithKeys("L"), key.WithHelp("L", "biblioteca")),
		Favorite:       key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "favorito")),
		AddToPlaylist:  key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "añadir a playlist")),
		CreatePlaylist: key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "crear playlist")),
	}
}
