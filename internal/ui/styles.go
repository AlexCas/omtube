package ui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

type styles struct {
	title     lipgloss.Style
	panel     lipgloss.Style
	heading   lipgloss.Style
	selected  lipgloss.Style
	current   lipgloss.Style
	dim       lipgloss.Style
	help      lipgloss.Style
	errorMsg  lipgloss.Style
	viz       lipgloss.Style
	sidebar   lipgloss.Style // caja de la columna lateral de altura completa (design D7a)
	card      lipgloss.Style // tarjeta de "ahora suena" al pie (design D7a)
	navActive lipgloss.Style // ítem de navegación activo: acento en negrita (design D7b)
	navItem   lipgloss.Style // ítem de navegación inactivo: apagado (design D7b)
	accentBar lipgloss.Style // barra/regla de acento de los encabezados (design D7c)
}

func defaultStyles() styles {
	return styles{
		title: lipgloss.NewStyle().Bold(true).
			Foreground(lipgloss.Color("#e0aaff")).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#e0aaff")).
			Padding(0, 1),
		panel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#e0aaff")).
			Padding(0, 1),
		heading:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#e0aaff")),
		selected: lipgloss.NewStyle().Foreground(lipgloss.Color("#00f5d4")).Bold(true),
		current:  lipgloss.NewStyle().Foreground(lipgloss.Color("#00f5d4")).Bold(true),
		dim:      lipgloss.NewStyle().Foreground(lipgloss.Color("#a0a0a0")),
		help:     lipgloss.NewStyle().Foreground(lipgloss.Color("#a0a0a0")),
		errorMsg: lipgloss.NewStyle().Foreground(lipgloss.Color("#e0aaff")).Bold(true),
		viz:      lipgloss.NewStyle().Foreground(lipgloss.Color("#e0aaff")),
		// Estilos del rediseño sidebar (design D7): todos translúcidos —
		// solo foreground/borde, NUNCA Background, para conservar el vidrio
		// de la terminal.
		sidebar: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#e0aaff")).
			Padding(0, 1),
		card: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#e0aaff")).
			Padding(0, 1),
		navActive: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#e0aaff")).
			Bold(true),
		navItem: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#a0a0a0")),
		accentBar: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#e0aaff")),
	}
}

// caelestiaListDelegate construye el delegate de bubbles/list con los acentos
// Caelestia para los modales (modeResults y pickers). La selección se distingue
// por foreground turquesa en negrita y un borde izquierdo mauve — nunca por un
// relleno de fondo opaco: ningún subestilo define Background, preservando la
// translucidez del vidrio de la terminal.
func caelestiaListDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()
	d.Styles.NormalTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a0a0a0")).
		Padding(0, 0, 0, 2)
	d.Styles.NormalDesc = d.Styles.NormalTitle.
		Foreground(lipgloss.Color("#a0a0a0"))
	d.Styles.SelectedTitle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00f5d4")).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("#e0aaff")).
		Padding(0, 0, 0, 1)
	d.Styles.SelectedDesc = d.Styles.SelectedTitle.
		Foreground(lipgloss.Color("#e0aaff"))
	d.Styles.DimmedTitle = d.Styles.NormalTitle
	d.Styles.DimmedDesc = d.Styles.NormalDesc
	return d
}
