// Package presence publica la pista en reproducción como presencia "escuchando"
// en Discord mediante su IPC local. Es una feature opcional que falla en
// silencio: si no hay app_id, Discord no está corriendo o el socket no responde,
// se convierte en un no-op registrado una sola vez, sin bloquear ni romper la
// reproducción.
package presence

import (
	"sync"

	"github.com/hugolgst/rich-go/client"
	"go.uber.org/zap"
)

// connector abstrae la conexión y el envío a Discord para poder inyectar un
// doble de prueba sin un Discord real. login devuelve error si la IPC falla;
// setActivity publica la actividad; logout cierra la conexión.
type connector interface {
	login(appID string) error
	setActivity(client.Activity) error
	logout()
}

// richGoConnector es el conector real respaldado por rich-go (IPC por socket
// Unix, pure Go).
type richGoConnector struct{}

func (richGoConnector) login(appID string) error            { return client.Login(appID) }
func (richGoConnector) setActivity(a client.Activity) error { return client.SetActivity(a) }
func (richGoConnector) logout()                             { client.Logout() }

// Client gestiona la presencia de Discord. Todas las operaciones son seguras de
// llamar aunque la conexión nunca se haya establecido: simplemente no hacen
// nada. El app_id es provisto por el usuario; no se incluye ninguno por
// defecto.
type Client struct {
	appID string
	conn  connector
	log   *zap.Logger

	mu        sync.Mutex
	connected bool
	warned    bool // garantiza que el aviso de fallo se registre una sola vez
}

// New construye un cliente de presencia para el app_id dado. Un app_id vacío
// deja la feature desactivada (Connect será un no-op silencioso). logger puede
// ser nil.
func New(appID string, logger *zap.Logger) *Client {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Client{appID: appID, conn: richGoConnector{}, log: logger}
}

// newWithConnector es el constructor testeable: permite inyectar un conector
// falso (p. ej. uno cuyo login siempre falla).
func newWithConnector(appID string, conn connector, logger *zap.Logger) *Client {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Client{appID: appID, conn: conn, log: logger}
}

// Connect intenta abrir la IPC con Discord. Es un no-op silencioso cuando el
// app_id está vacío (feature desactivada) o cuando la conexión falla (Discord
// cerrado / socket ausente). Nunca devuelve error: el aviso se registra como
// máximo una vez para no inundar el log.
func (c *Client) Connect() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.appID == "" {
		c.warnOnce("discord presence disabled: no app_id configured")
		return
	}
	if c.connected {
		return
	}
	if err := c.conn.login(c.appID); err != nil {
		c.warnOnce("discord presence unavailable", zap.Error(err))
		return
	}
	c.connected = true
}

// Set publica la pista actual como actividad "escuchando". Es un no-op si la
// presencia no está conectada. Los errores de envío se ignoran para no afectar
// la reproducción.
func (c *Client) Set(title, artist string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.connected {
		return
	}
	_ = c.conn.setActivity(client.Activity{
		Details: title,
		State:   artist,
	})
}

// Clear limpia la actividad publicada (al detener la reproducción). Es un no-op
// si no hay conexión.
func (c *Client) Clear() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.connected {
		return
	}
	_ = c.conn.setActivity(client.Activity{})
}

// Close cierra la conexión IPC. Seguro de llamar aunque nunca se haya
// conectado.
func (c *Client) Close() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.connected {
		return
	}
	c.conn.logout()
	c.connected = false
}

// warnOnce registra un aviso a nivel warn una única vez por cliente. Debe
// llamarse con el mutex tomado.
func (c *Client) warnOnce(msg string, fields ...zap.Field) {
	if c.warned {
		return
	}
	c.warned = true
	c.log.Warn(msg, fields...)
}
