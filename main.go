// Command terminaltube es un reproductor de música TUI que usa YouTube vía yt-dlp y
// mpv.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"go.uber.org/zap"

	"github.com/alexcasdev/terminaltube/internal/artwork"
	"github.com/alexcasdev/terminaltube/internal/cache"
	"github.com/alexcasdev/terminaltube/internal/config"
	"github.com/alexcasdev/terminaltube/internal/favorites"
	"github.com/alexcasdev/terminaltube/internal/history"
	"github.com/alexcasdev/terminaltube/internal/logging"
	"github.com/alexcasdev/terminaltube/internal/lyrics"
	"github.com/alexcasdev/terminaltube/internal/metadata"
	"github.com/alexcasdev/terminaltube/internal/mpris"
	"github.com/alexcasdev/terminaltube/internal/player"
	"github.com/alexcasdev/terminaltube/internal/playlist"
	"github.com/alexcasdev/terminaltube/internal/presence"
	"github.com/alexcasdev/terminaltube/internal/search"
	"github.com/alexcasdev/terminaltube/internal/storage"
	"github.com/alexcasdev/terminaltube/internal/ui"
)

// version es la versión del binario. Se sobreescribe en el build del release vía
// -ldflags (GoReleaser); en builds locales queda como "dev".
var version = "dev"

func main() {
	for _, arg := range os.Args[1:] {
		if arg == "--version" || arg == "-v" {
			fmt.Println("omusic", version)
			return
		}
	}
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "omusic:", err)
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

	svc, closePresence := buildServices(cfg, db, logger)
	defer closePresence()

	model := ui.New(cfg, searcher, mpv, hist, playlistSvc, favoritesSvc, svc, logger)
	prog := tea.NewProgram(&model, tea.WithAltScreen())

	mprisSvc := mpris.New(func(msg interface{}) { prog.Send(msg) }, logger)
	model.SetMpris(mprisSvc)
	if mprisSvc != nil {
		defer mprisSvc.Close()
	}

	_, err = prog.Run()
	return err
}

// buildServices construye los servicios de enriquecimiento opcionales según los
// toggles de configuración. Cualquier servicio desactivado queda en nil, de modo
// que con todos los toggles apagados la UI se comporta como en la Fase 2.
// Devuelve también un cierre que libera la presencia de Discord al salir.
func buildServices(cfg config.Config, db *storage.DB, logger *zap.Logger) (ui.Services, func()) {
	var svc ui.Services
	noop := func() {}

	if cfg.CacheEnabled {
		cacheSvc := cache.New(
			db.Cache(),
			cfg.YtDlpPath,
			cfg.CacheDir(),
			int64(cfg.CacheMaxSizeMB)*1024*1024,
			time.Duration(cfg.CacheMaxAgeDays)*24*time.Hour,
		)
		// Sweep de arranque: recupera espacio entre sesiones y tras bajar límites.
		if err := cacheSvc.Sweep(); err != nil {
			logger.Warn("sweep de caché al arrancar falló: " + err.Error())
		}
		svc.Cache = cacheSvc
	}

	if cfg.LyricsEnabled {
		lyricsSvc := lyrics.New(db.Lyrics(), &http.Client{Timeout: 10 * time.Second})
		// Refleja el toggle de configuración en el servicio (WU1 dejó este seam).
		lyricsSvc.SetSearchFallback(cfg.LyricsSearchFallback)
		svc.Lyrics = lyricsSvc
	}

	if cfg.ArtworkEnabled {
		// Reutiliza la miniatura cacheada por yt-dlp (--write-thumbnail) cuando
		// existe; solo cae a la URL remota de YouTube ante un miss. El resolutor
		// es nil-safe: con la caché desactivada siempre usa la URL remota.
		var thumb func(string) (string, bool)
		if cs, ok := svc.Cache.(*cache.Service); ok {
			thumb = cs.ThumbPath
		}
		// El resolutor de portada real (MusicBrainz + Cover Art Archive) solo se
		// construye con el toggle activo; en nil el adaptador es byte-idéntico a
		// la Fase 3 (solo thumbnail).
		var cover artwork.CoverResolver
		if cfg.ArtworkCoverArt {
			cover = artwork.NewCoverResolver(cfg.CacheDir(), &http.Client{Timeout: 10 * time.Second})
		}
		svc.Artwork = artworkAdapter{backend: artwork.Detect(), thumb: thumb, cover: cover}
	}

	if cfg.PresenceActive() {
		pres := presence.New(cfg.PresenceAppID, logger)
		pres.Connect()
		svc.Presence = pres
		return svc, pres.Close
	}

	return svc, noop
}

// artworkAdapter adapta artwork.Backend (que renderiza desde una ruta o URL de
// miniatura) al contrato artworkService de la UI. Prefiere la miniatura cacheada
// localmente (descargada con el audio vía --write-thumbnail) y solo deriva la
// URL remota de YouTube cuando no hay copia local, conforme a la decisión de
// diseño "reuse cached thumbnail".
type artworkAdapter struct {
	backend artwork.Backend
	thumb   func(id string) (string, bool) // resolutor de miniatura local; nil ⇒ siempre remota
	cover   artwork.CoverResolver          // resolutor de portada real; nil ⇒ solo thumbnail (Fase 3)
}

// Render selecciona la fuente de portada con la precedencia: portada real
// resuelta (MusicBrainz + Cover Art Archive) → miniatura cacheada localmente →
// URL remota de YouTube. La resolución de red vive aquí, no en Backend.Render,
// que se mantiene puro. Con cover=nil el comportamiento es byte-idéntico a la
// Fase 3.
func (a artworkAdapter) Render(ctx context.Context, track search.Result, w, h int) string {
	src := "https://i.ytimg.com/vi/" + track.ID + "/hqdefault.jpg"
	if a.thumb != nil {
		if local, ok := a.thumb(track.ID); ok {
			src = local
		}
	}
	if a.cover != nil {
		artist, title := metadata.Normalize(track)
		if path, ok := a.cover.Resolve(ctx, artist, title, track.Duration); ok {
			src = path
		}
	}
	out, _ := a.backend.Render(ctx, src, w, h)
	return out
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
