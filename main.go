// Command terminaltube es un reproductor de música TUI que usa YouTube vía yt-dlp y
// mpv.
package main

import (
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/alexcasdev/terminaltube/internal/config"
	"github.com/alexcasdev/terminaltube/internal/favorites"
	"github.com/alexcasdev/terminaltube/internal/history"
	"github.com/alexcasdev/terminaltube/internal/logging"
	"github.com/alexcasdev/terminaltube/internal/player"
	"github.com/alexcasdev/terminaltube/internal/playlist"
	"github.com/alexcasdev/terminaltube/internal/search"
	"github.com/alexcasdev/terminaltube/internal/storage"
	"github.com/alexcasdev/terminaltube/internal/ui"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "terminaltube:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("cargando configuración: %w", err)
	}

	if err := checkDeps(cfg); err != nil {
		return err
	}

	logger := logging.New(cfg.LogFile())
	defer logger.Sync()

	db, err := storage.Open(cfg.LibraryFile())
	if err != nil {
		return fmt.Errorf("abriendo biblioteca: %w", err)
	}
	defer db.Close()

	tracksRepo := db.Tracks()
	playlistSvc := playlist.New(db.Playlists(), tracksRepo)
	favoritesSvc := favorites.New(db.Favorites(), tracksRepo)

	hist, err := history.Load(db.History(), tracksRepo, cfg.HistoryFile())
	if err != nil {
		return fmt.Errorf("cargando historial: %w", err)
	}

	searcher := search.NewYtDlp(cfg.YtDlpPath)

	mpv, err := player.NewMPV(cfg.MpvPath, cfg.SocketPath(), cfg.Volume)
	if err != nil {
		return fmt.Errorf("iniciando mpv: %w", err)
	}
	defer mpv.Close()

	model := ui.New(cfg, searcher, mpv, hist, playlistSvc, favoritesSvc, logger)
	prog := tea.NewProgram(model, tea.WithAltScreen())
	_, err = prog.Run()
	return err
}

// checkDeps valida que yt-dlp y mpv estén disponibles en PATH.
func checkDeps(cfg config.Config) error {
	for _, bin := range []string{cfg.YtDlpPath, cfg.MpvPath} {
		if _, err := exec.LookPath(bin); err != nil {
			return fmt.Errorf("dependencia no encontrada en PATH: %q (instala yt-dlp y mpv)", bin)
		}
	}
	return nil
}
