// Package config carga la configuración de TerminalTube vía Viper y resuelve las
// rutas XDG donde se guardan config, datos y logs.
package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config contiene los ajustes de la aplicación.
type Config struct {
	// SearchResults es el número máximo de resultados que devuelve una búsqueda.
	SearchResults int
	// Volume es el volumen inicial de mpv (0–130).
	Volume int
	// MpvPath y YtDlpPath permiten sobreescribir los binarios usados.
	MpvPath   string
	YtDlpPath string

	// Caché de audio: descarga las pistas a disco para reproducirlas sin volver
	// a streamear. CacheMaxSizeMB y CacheMaxAgeDays acotan el tamaño/antigüedad
	// (<=0 desactiva esa dimensión).
	CacheEnabled    bool
	CacheMaxSizeMB  int
	CacheMaxAgeDays int

	// LyricsEnabled activa el panel de letra (lrclib).
	LyricsEnabled bool
	// ArtworkEnabled activa el panel de portada.
	ArtworkEnabled bool

	// PresenceEnabled activa la presencia de Discord. Permanece inactiva mientras
	// PresenceAppID esté vacío (el usuario debe proveer su propio app_id).
	PresenceEnabled bool
	PresenceAppID   string

	// Rutas resueltas (no provienen del archivo de config).
	ConfigDir string
	DataDir   string
	StateDir  string
	CacheHome string
}

// HistoryFile devuelve la ruta del archivo JSON de historial (legado, usado solo
// para la importación única hacia SQLite).
func (c Config) HistoryFile() string { return filepath.Join(c.DataDir, "history.json") }

// LibraryFile devuelve la ruta de la base de datos SQLite de la biblioteca
// (playlists, favoritos e historial). Es una ruta XDG fija, sin override por
// configuración.
func (c Config) LibraryFile() string { return filepath.Join(c.DataDir, "library.db") }

// LogFile devuelve la ruta del archivo de logs.
func (c Config) LogFile() string { return filepath.Join(c.StateDir, "terminaltube.log") }

// CacheDir devuelve el directorio raíz de la caché de audio (XDG cache).
func (c Config) CacheDir() string { return c.CacheHome }

// PresenceActive indica si la presencia de Discord debe ejecutarse: requiere el
// toggle activo Y un app_id provisto por el usuario.
func (c Config) PresenceActive() bool { return c.PresenceEnabled && c.PresenceAppID != "" }

// SocketPath devuelve la ruta del socket IPC de mpv.
func (c Config) SocketPath() string {
	if rt := os.Getenv("XDG_RUNTIME_DIR"); rt != "" {
		return filepath.Join(rt, "terminaltube-mpv.sock")
	}
	return filepath.Join(os.TempDir(), "terminaltube-mpv.sock")
}

// Load lee la configuración aplicando defaults y, si existe,
// ~/.config/terminaltube/config.yaml. También crea los directorios necesarios.
func Load() (Config, error) {
	configDir := xdgDir("XDG_CONFIG_HOME", ".config")
	dataDir := xdgDir("XDG_DATA_HOME", ".local/share")
	stateDir := xdgDir("XDG_STATE_HOME", ".local/state")
	cacheDir := xdgDir("XDG_CACHE_HOME", ".cache")

	appConfigDir := filepath.Join(configDir, "terminaltube")
	appDataDir := filepath.Join(dataDir, "terminaltube")
	appStateDir := filepath.Join(stateDir, "terminaltube")
	appCacheDir := filepath.Join(cacheDir, "terminaltube")

	v := viper.New()
	v.SetDefault("search_results", 10)
	v.SetDefault("volume", 70)
	v.SetDefault("mpv_path", "mpv")
	v.SetDefault("ytdlp_path", "yt-dlp")

	// Defaults de enriquecimiento (Fase 3): caché/letra/portada activas; la
	// presencia de Discord queda inactiva hasta que el usuario provea app_id.
	v.SetDefault("cache.enabled", true)
	v.SetDefault("cache.max_size_mb", 1024)
	v.SetDefault("cache.max_age_days", 30)
	v.SetDefault("lyrics.enabled", true)
	v.SetDefault("artwork.enabled", true)
	v.SetDefault("presence.enabled", false)
	v.SetDefault("presence.app_id", "")

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(appConfigDir)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return Config{}, err
		}
		// Sin archivo de config: se usan los defaults.
	}

	for _, d := range []string{appConfigDir, appDataDir, appStateDir, appCacheDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return Config{}, err
		}
	}

	return Config{
		SearchResults: v.GetInt("search_results"),
		Volume:        v.GetInt("volume"),
		MpvPath:       v.GetString("mpv_path"),
		YtDlpPath:     v.GetString("ytdlp_path"),

		CacheEnabled:    v.GetBool("cache.enabled"),
		CacheMaxSizeMB:  v.GetInt("cache.max_size_mb"),
		CacheMaxAgeDays: v.GetInt("cache.max_age_days"),
		LyricsEnabled:   v.GetBool("lyrics.enabled"),
		ArtworkEnabled:  v.GetBool("artwork.enabled"),
		PresenceEnabled: v.GetBool("presence.enabled"),
		PresenceAppID:   v.GetString("presence.app_id"),

		ConfigDir: appConfigDir,
		DataDir:   appDataDir,
		StateDir:  appStateDir,
		CacheHome: appCacheDir,
	}, nil
}

func xdgDir(env, fallback string) string {
	if v := os.Getenv(env); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return fallback
	}
	return filepath.Join(home, fallback)
}
