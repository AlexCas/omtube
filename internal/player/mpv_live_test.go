//go:build live

package player

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestMPVIPCHandshake valida contra un mpv real el arranque idle, la conexión al
// socket y el ciclo comando/respuesta (set/get volumen). No requiere red ni audio.
// Ejecutar con: go test -tags live ./internal/player/
func TestMPVIPCHandshake(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "mpv.sock")
	m, err := NewMPV("mpv", sock, 70)
	if err != nil {
		t.Fatalf("NewMPV: %v", err)
	}
	defer m.Close()

	if got := m.Volume(); got != 70 {
		t.Fatalf("volumen inicial = %d, want 70", got)
	}
	v, err := m.AddVolume(10)
	if err != nil {
		t.Fatalf("AddVolume: %v", err)
	}
	if v != 80 {
		t.Fatalf("volumen tras +10 = %d, want 80", v)
	}
	// get_property real desde mpv confirma el round-trip por el socket.
	data, err := m.command("get_property", "volume")
	if err != nil {
		t.Fatalf("get_property volume: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("respuesta vacía de mpv")
	}

	if err := m.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	if _, err := os.Stat(sock); !os.IsNotExist(err) {
		t.Errorf("el socket no se eliminó tras Close")
	}
}
