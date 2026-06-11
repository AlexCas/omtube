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

	// Rutas resueltas (no provienen del archivo de config).
	ConfigDir string
	DataDir   string
	StateDir  string
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

	appConfigDir := filepath.Join(configDir, "terminaltube")
	appDataDir := filepath.Join(dataDir, "terminaltube")
	appStateDir := filepath.Join(stateDir, "terminaltube")

	v := viper.New()
	v.SetDefault("search_results", 10)
	v.SetDefault("volume", 70)
	v.SetDefault("mpv_path", "mpv")
	v.SetDefault("ytdlp_path", "yt-dlp")

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(appConfigDir)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return Config{}, err
		}
		// Sin archivo de config: se usan los defaults.
	}

	for _, d := range []string{appConfigDir, appDataDir, appStateDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return Config{}, err
		}
	}

	return Config{
		SearchResults: v.GetInt("search_results"),
		Volume:        v.GetInt("volume"),
		MpvPath:       v.GetString("mpv_path"),
		YtDlpPath:     v.GetString("ytdlp_path"),
		ConfigDir:     appConfigDir,
		DataDir:       appDataDir,
		StateDir:      appStateDir,
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
