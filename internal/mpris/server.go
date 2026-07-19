// Package mpris expone a Omusic como reproductor MPRIS v2 en el bus de sesión
// D-Bus. Es una feature opcional: si el bus no está disponible, el constructor
// devuelve nil y Omusic sigue funcionando normalmente. Los manejadores D-Bus
// nunca tocan el reproductor ni la cola; emiten tea.Msg via prog.Send para que
// la UI los procese en su goroutine principal.
package mpris

import (
	"sync"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/prop"
	"go.uber.org/zap"

	"github.com/alexcasdev/terminaltube/internal/lyrics"
	"github.com/alexcasdev/terminaltube/internal/search"
)

// Message types emitted by D-Bus handlers into the Bubble Tea event loop.
type PlayPauseMsg struct{}
type NextMsg struct{}
type PrevMsg struct{}
type StopMsg struct{}
type SeekMsg struct{ Offset int64 } // µs
type SetVolumeMsg struct{ Volume float64 }

const (
	dbusName    = "org.mpris.MediaPlayer2.omusic"
	dbusPath    = dbus.ObjectPath("/org/mpris/MediaPlayer2")
	playerIface = "org.mpris.MediaPlayer2.Player"
	rootIface   = "org.mpris.MediaPlayer2"
)

// Server implements the MPRIS v2 D-Bus interfaces for Omusic.
type Server struct {
	send func(interface{})
	log  *zap.Logger

	mu     sync.Mutex
	conn   *dbus.Conn
	props  *prop.Properties
	closed bool
}

// New registers an MPRIS v2 server on the session bus. If the session bus is
// unavailable, it logs a warning and returns (nil, nil) so the caller can keep
// running without MPRIS.
func New(send func(interface{}), logger *zap.Logger) *Server {
	if logger == nil {
		logger = zap.NewNop()
	}
	if send == nil {
		send = func(interface{}) {}
	}

	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		logger.Warn("mpris no disponible: no se pudo conectar al bus de sesión", zap.Error(err))
		return nil
	}

	s := newServer(send, logger)
	s.conn = conn

	propsSpec := prop.Map{
		rootIface: {
			"Identity":            {Value: "Omusic", Writable: false, Emit: prop.EmitFalse},
			"DesktopEntry":        {Value: "omusic", Writable: false, Emit: prop.EmitFalse},
			"CanQuit":             {Value: true, Writable: false, Emit: prop.EmitFalse},
			"CanRaise":            {Value: false, Writable: false, Emit: prop.EmitFalse},
			"HasTrackList":        {Value: false, Writable: false, Emit: prop.EmitFalse},
			"SupportedUriSchemes": {Value: []string{}, Writable: false, Emit: prop.EmitFalse},
			"SupportedMimeTypes":  {Value: []string{}, Writable: false, Emit: prop.EmitFalse},
		},
		playerIface: {
			"PlaybackStatus": {Value: "Stopped", Writable: true, Emit: prop.EmitTrue},
			"Metadata":       {Value: map[string]dbus.Variant{}, Writable: true, Emit: prop.EmitTrue},
			"Volume":         {Value: 1.0, Writable: true, Emit: prop.EmitTrue},
			"Position":       {Value: int64(0), Writable: false, Emit: prop.EmitFalse},
			"Rate":           {Value: 1.0, Writable: true, Emit: prop.EmitTrue},
			"LoopStatus":     {Value: "None", Writable: true, Emit: prop.EmitTrue},
			"MinimumRate":    {Value: 1.0, Writable: false, Emit: prop.EmitFalse},
			"MaximumRate":    {Value: 1.0, Writable: false, Emit: prop.EmitFalse},
			"CanGoNext":      {Value: true, Writable: false, Emit: prop.EmitFalse},
			"CanGoPrevious":  {Value: true, Writable: false, Emit: prop.EmitFalse},
			"CanPlay":        {Value: true, Writable: false, Emit: prop.EmitFalse},
			"CanPause":       {Value: true, Writable: false, Emit: prop.EmitFalse},
			"CanSeek":        {Value: true, Writable: false, Emit: prop.EmitFalse},
			"CanControl":     {Value: true, Writable: false, Emit: prop.EmitFalse},
		},
	}

	props, err := prop.Export(conn, dbusPath, propsSpec)
	if err != nil {
		_ = conn.Close()
		logger.Warn("mpris no disponible: no se pudieron exportar propiedades", zap.Error(err))
		return nil
	}
	s.props = props

	if err := conn.Export(root{s}, dbusPath, rootIface); err != nil {
		_ = conn.Close()
		logger.Warn("mpris no disponible: no se pudo exportar la interfaz raíz", zap.Error(err))
		return nil
	}
	p := player{s}
	if err := conn.ExportWithMap(p, map[string]string{"SeekRelative": "Seek"}, dbusPath, playerIface); err != nil {
		_ = conn.Close()
		logger.Warn("mpris no disponible: no se pudo exportar la interfaz de reproductor", zap.Error(err))
		return nil
	}

	reply, err := conn.RequestName(dbusName, dbus.NameFlagDoNotQueue)
	if err != nil {
		_ = conn.Close()
		logger.Warn("mpris no disponible: no se pudo solicitar el nombre", zap.Error(err))
		return nil
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		_ = conn.Close()
		logger.Warn("mpris no disponible: otro proceso ya posee el nombre", zap.String("name", dbusName))
		return nil
	}

	return s
}

func newServer(send func(interface{}), logger *zap.Logger) *Server {
	return &Server{
		send: send,
		log:  logger,
	}
}

// SetMetadata updates the Metadata property with track and optional lyrics.
func (s *Server) SetMetadata(track search.Result, lyrics lyrics.Lyrics) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.props == nil {
		return
	}
	s.props.SetMust(playerIface, "Metadata", metadataDict(track, lyrics))
}

// SetPlaybackStatus updates the PlaybackStatus property.
func (s *Server) SetPlaybackStatus(status string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.props == nil {
		return
	}
	s.props.SetMust(playerIface, "PlaybackStatus", status)
}

// SetVolume updates the Volume property. vol is in the player scale 0–130.
func (s *Server) SetVolume(vol int) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.props == nil {
		return
	}
	s.props.SetMust(playerIface, "Volume", float64(vol)/130.0)
}

// SetPosition updates the Position property. pos is in seconds.
func (s *Server) SetPosition(pos float64) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.props == nil {
		return
	}
	s.props.SetMust(playerIface, "Position", int64(pos*1e6))
}

// Seeked emits the Seeked signal with the given position in microseconds.
func (s *Server) Seeked(positionUS int64) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.conn == nil {
		return
	}
	if err := s.conn.Emit(dbusPath, playerIface+".Seeked", positionUS); err != nil && s.log != nil {
		s.log.Warn("no se pudo emitir señal Seeked", zap.Error(err))
	}
}

// Close releases the D-Bus name and closes the connection.
func (s *Server) Close() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.conn == nil {
		return
	}
	_, _ = s.conn.ReleaseName(dbusName)
	_ = s.conn.Close()
	s.closed = true
}

// metadataDict maps a search.Result and optional lyrics to an MPRIS Metadata dict.
func metadataDict(track search.Result, lyrics lyrics.Lyrics) map[string]dbus.Variant {
	m := map[string]dbus.Variant{
		"xesam:title":  dbus.MakeVariant(track.Title),
		"xesam:artist": dbus.MakeVariant(track.Uploader),
		"xesam:album":  dbus.MakeVariant(""),
		"mpris:length": dbus.MakeVariant(int64(track.Duration) * 1e6),
		"mpris:artUrl": dbus.MakeVariant("https://i.ytimg.com/vi/" + track.ID + "/hqdefault.jpg"),
	}
	if lyrics.Synced {
		lines := make([]string, len(lyrics.Lines))
		for i, l := range lyrics.Lines {
			lines[i] = l.Text
		}
		m["xesam:asText"] = dbus.MakeVariant(lines)
	}
	return m
}

// root implements org.mpris.MediaPlayer2 methods.
type root struct{ s *Server }

func (r root) Raise() *dbus.Error { return nil }
func (r root) Quit() *dbus.Error  { return nil }

// player implements org.mpris.MediaPlayer2.Player methods.
type player struct{ s *Server }

func (p player) Next() *dbus.Error        { p.s.send(NextMsg{}); return nil }
func (p player) Previous() *dbus.Error     { p.s.send(PrevMsg{}); return nil }
func (p player) Pause() *dbus.Error       { p.s.send(PlayPauseMsg{}); return nil }
func (p player) PlayPause() *dbus.Error    { p.s.send(PlayPauseMsg{}); return nil }
func (p player) Stop() *dbus.Error         { p.s.send(StopMsg{}); return nil }
func (p player) Play() *dbus.Error        { p.s.send(PlayPauseMsg{}); return nil }
// SeekRelative implements the MPRIS Seek method. It is exported as "Seek" on
// D-Bus via ExportWithMap; the Go method name differs to avoid colliding with
// the io.Seeker signature checked by go vet.
func (p player) SeekRelative(offset int64) *dbus.Error {
	p.s.send(SeekMsg{Offset: offset})
	return nil
}
func (p player) SetVolume(volume float64) *dbus.Error {
	p.s.send(SetVolumeMsg{Volume: volume})
	return nil
}
func (p player) OpenUri(_ string) *dbus.Error { return nil }
