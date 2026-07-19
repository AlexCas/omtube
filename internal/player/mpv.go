package player

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/alexcasdev/terminaltube/internal/search"
)

// MPV controla un proceso mpv en modo idle a través de su socket IPC JSON.
type MPV struct {
	cmd    *exec.Cmd
	socket string

	mu      sync.Mutex
	conn    net.Conn
	nextID  int
	pending map[int]chan rawResponse
	paused  bool
	volume  int

	events chan Event
	closed bool
}

type rawResponse struct {
	Data  json.RawMessage `json:"data"`
	Error string          `json:"error"`
}

// ipcMessage es un mensaje recibido por el socket (respuesta o evento).
type ipcMessage struct {
	RequestID *int            `json:"request_id"`
	Error     string          `json:"error"`
	Data      json.RawMessage `json:"data"`
	Event     string          `json:"event"`
	Reason    string          `json:"reason"`
}

// NewMPV lanza mpv y se conecta a su socket IPC. initialVolume se aplica al iniciar.
func NewMPV(bin, socket string, initialVolume int) (*MPV, error) {
	if bin == "" {
		bin = "mpv"
	}
	// Un socket previo huérfano impediría que mpv arranque su servidor IPC.
	_ = os.Remove(socket)

	cmd := exec.Command(bin,
		"--idle=yes",
		"--no-video",
		"--no-terminal",
		"--ytdl-format=bestaudio/best",
		"--input-ipc-server="+socket,
	)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("no se pudo iniciar mpv: %w", err)
	}

	m := &MPV{
		cmd:     cmd,
		socket:  socket,
		pending: make(map[int]chan rawResponse),
		volume:  clampVolume(initialVolume),
		events:  make(chan Event, 16),
	}

	conn, err := dialWithRetry(socket, 3*time.Second)
	if err != nil {
		_ = cmd.Process.Kill()
		return nil, fmt.Errorf("no se pudo conectar al socket de mpv: %w", err)
	}
	m.conn = conn

	go m.readLoop()

	_, _ = m.command("set_property", "volume", m.volume)
	return m, nil
}

func dialWithRetry(socket string, timeout time.Duration) (net.Conn, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		conn, err := dialIPC(socket, 50*time.Millisecond)
		if err == nil {
			return conn, nil
		}
		lastErr = err
		time.Sleep(50 * time.Millisecond)
	}
	return nil, lastErr
}

// readLoop lee mensajes del socket y los despacha a respuestas o eventos.
func (m *MPV) readLoop() {
	sc := bufio.NewScanner(m.conn)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var msg ipcMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}
		switch {
		case msg.RequestID != nil:
			m.mu.Lock()
			ch, ok := m.pending[*msg.RequestID]
			if ok {
				delete(m.pending, *msg.RequestID)
			}
			m.mu.Unlock()
			if ok {
				ch <- rawResponse{Data: msg.Data, Error: msg.Error}
			}
		case msg.Event == "end-file":
			// Solo el fin natural debe avanzar la cola; "stop"/"redirect" provienen
			// de un loadfile nuestro.
			if msg.Reason == "eof" {
				m.emit(Event{Kind: EventEndFile})
			}
		}
	}
}

func (m *MPV) emit(e Event) {
	select {
	case m.events <- e:
	default: // si nadie consume, se descarta para no bloquear
	}
}

// command envía un comando JSON y espera la respuesta correlacionada por request_id.
func (m *MPV) command(args ...interface{}) (json.RawMessage, error) {
	m.mu.Lock()
	if m.closed || m.conn == nil {
		m.mu.Unlock()
		return nil, fmt.Errorf("mpv cerrado")
	}
	m.nextID++
	id := m.nextID
	ch := make(chan rawResponse, 1)
	m.pending[id] = ch

	payload := map[string]interface{}{"command": args, "request_id": id}
	data, err := json.Marshal(payload)
	if err != nil {
		delete(m.pending, id)
		m.mu.Unlock()
		return nil, err
	}
	data = append(data, '\n')
	_, werr := m.conn.Write(data)
	m.mu.Unlock()
	if werr != nil {
		return nil, werr
	}

	select {
	case resp := <-ch:
		if resp.Error != "" && resp.Error != "success" {
			return nil, fmt.Errorf("mpv: %s", resp.Error)
		}
		return resp.Data, nil
	case <-time.After(2 * time.Second):
		m.mu.Lock()
		delete(m.pending, id)
		m.mu.Unlock()
		return nil, fmt.Errorf("timeout esperando a mpv")
	}
}

// Load reproduce la fuente indicada: una ruta de archivo local cacheado o una
// URL/ID de YouTube (mpv resuelve el audio remoto con su hook yt-dlp). Para
// ambos casos basta loadfile, que acepta tanto rutas locales como URLs.
func (m *MPV) Load(src string) error {
	if _, err := m.command("loadfile", src, "replace"); err != nil {
		return err
	}
	m.mu.Lock()
	m.paused = false
	m.mu.Unlock()
	_, _ = m.command("set_property", "pause", false)
	m.emit(Event{Kind: EventLoaded})
	return nil
}

// LoadTrack carga la fuente src (archivo local o URL de YouTube) y, al iniciar
// la nueva pista, emite EventTrackChange con su metadato para que la UI dispare
// la obtención de letra/portada y actualice la presencia de Discord.
func (m *MPV) LoadTrack(src string, track search.Result) error {
	if err := m.Load(src); err != nil {
		return err
	}
	m.emit(Event{Kind: EventTrackChange, Track: track, Source: src})
	return nil
}

// Stop detiene la reproducción y vacía la lista interna de mpv mediante el comando
// `stop`, dejando el proceso vivo y ocioso para reproducir más tarde. El estado de
// pausa se restablece a falso.
func (m *MPV) Stop() error {
	if _, err := m.command("stop"); err != nil {
		return err
	}
	m.mu.Lock()
	m.paused = false
	m.mu.Unlock()
	return nil
}

// TogglePause alterna pausa/reproducción y sincroniza el estado real desde mpv.
func (m *MPV) TogglePause() error {
	if _, err := m.command("cycle", "pause"); err != nil {
		return err
	}
	// Leer el estado real evita que m.paused se desincronice de mpv (p.ej. tras
	// fin de pista o un cycle sobre mpv en idle).
	if data, err := m.command("get_property", "pause"); err == nil {
		var paused bool
		if json.Unmarshal(data, &paused) == nil {
			m.mu.Lock()
			m.paused = paused
			m.mu.Unlock()
		}
	}
	return nil
}

// AddVolume ajusta el volumen (clamp 0–130) y devuelve el nuevo valor.
func (m *MPV) AddVolume(delta int) (int, error) {
	m.mu.Lock()
	m.volume = clampVolume(m.volume + delta)
	v := m.volume
	m.mu.Unlock()
	_, err := m.command("set_property", "volume", v)
	return v, err
}

// Seek salta offset segundos de forma relativa a la posición actual.
func (m *MPV) Seek(offset float64) error {
	_, err := m.command("seek", offset, "relative")
	return err
}

// Position devuelve la posición y duración actuales (0 si no disponibles).
func (m *MPV) Position() (pos, dur float64) {
	pos = m.getFloat("time-pos")
	dur = m.getFloat("duration")
	return pos, dur
}

func (m *MPV) getFloat(prop string) float64 {
	data, err := m.command("get_property", prop)
	if err != nil {
		return 0
	}
	var f float64
	if err := json.Unmarshal(data, &f); err != nil {
		return 0
	}
	return f
}

// Paused indica si la reproducción está pausada.
func (m *MPV) Paused() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.paused
}

// Volume devuelve el volumen actual.
func (m *MPV) Volume() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.volume
}

// Events expone el canal de eventos.
func (m *MPV) Events() <-chan Event { return m.events }

// Close detiene mpv y cierra el socket.
func (m *MPV) Close() error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil
	}
	m.closed = true
	conn := m.conn
	m.conn = nil
	m.mu.Unlock()

	if conn != nil {
		_, _ = conn.Write([]byte(`{"command":["quit"]}` + "\n"))
		_ = conn.Close()
	}
	if m.cmd != nil && m.cmd.Process != nil {
		_ = m.cmd.Process.Kill()
		_, _ = m.cmd.Process.Wait()
	}
	_ = os.Remove(m.socket)
	return nil
}

func clampVolume(v int) int {
	if v < 0 {
		return 0
	}
	if v > 130 {
		return 130
	}
	return v
}
