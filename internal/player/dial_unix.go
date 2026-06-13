//go:build !windows

package player

import (
	"net"
	"time"
)

// dialIPC conecta al socket IPC de mpv. En sistemas Unix es un socket de dominio.
func dialIPC(socket string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout("unix", socket, timeout)
}
