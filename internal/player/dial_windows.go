//go:build windows

package player

import (
	"net"
	"time"

	"gopkg.in/natefinch/npipe.v2"
)

// dialIPC conecta al named pipe IPC de mpv en Windows.
func dialIPC(socket string, timeout time.Duration) (net.Conn, error) {
	return npipe.DialTimeout(socket, timeout)
}
