package presence

import (
	"errors"
	"testing"

	"github.com/hugolgst/rich-go/client"
)

// fakeConn es un conector falso que registra las llamadas y puede simular un
// fallo de login (Discord cerrado / socket ausente).
type fakeConn struct {
	loginErr     error
	logins       int
	activities   []client.Activity
	logouts      int
	failActivity bool
}

func (f *fakeConn) login(appID string) error {
	f.logins++
	return f.loginErr
}

func (f *fakeConn) setActivity(a client.Activity) error {
	f.activities = append(f.activities, a)
	if f.failActivity {
		return errors.New("boom")
	}
	return nil
}

func (f *fakeConn) logout() { f.logouts++ }

func TestConnectEmptyAppIDIsSilentNoOp(t *testing.T) {
	conn := &fakeConn{}
	c := newWithConnector("", conn, nil)

	c.Connect()
	c.Set("Song", "Artist")
	c.Clear()
	c.Close()

	if conn.logins != 0 {
		t.Fatalf("appID vacío no debe intentar login, logins=%d", conn.logins)
	}
	if len(conn.activities) != 0 {
		t.Fatalf("sin conexión no debe publicar actividad, got %d", len(conn.activities))
	}
	if conn.logouts != 0 {
		t.Fatalf("sin conexión no debe hacer logout, got %d", conn.logouts)
	}
	if c.connected {
		t.Fatalf("no debería estar conectado con appID vacío")
	}
}

func TestConnectFailingDialerIsSilentNoOp(t *testing.T) {
	conn := &fakeConn{loginErr: errors.New("dial unix: no such file")}
	c := newWithConnector("123456789", conn, nil)

	// No debe entrar en pánico ni propagar el error (Connect no devuelve error).
	c.Connect()
	if c.connected {
		t.Fatalf("login fallido no debe marcar conectado")
	}

	// Operaciones posteriores son no-ops seguros.
	c.Set("Song", "Artist")
	c.Clear()
	c.Close()

	if len(conn.activities) != 0 {
		t.Fatalf("login fallido no debe publicar actividad, got %d", len(conn.activities))
	}
	if conn.logouts != 0 {
		t.Fatalf("login fallido no debe hacer logout, got %d", conn.logouts)
	}
}

func TestConnectWarnsOnce(t *testing.T) {
	conn := &fakeConn{loginErr: errors.New("nope")}
	c := newWithConnector("123", conn, nil)

	c.Connect()
	c.Connect()
	c.Connect()

	if !c.warned {
		t.Fatalf("se esperaba que el aviso se registrara")
	}
	// Cada Connect reintenta el login (no estaba conectado), pero el aviso solo
	// se loguea una vez (warnOnce).
	if conn.logins != 3 {
		t.Fatalf("se esperaban 3 intentos de login, got %d", conn.logins)
	}
}

func TestHappyPathSetClearClose(t *testing.T) {
	conn := &fakeConn{}
	c := newWithConnector("appid", conn, nil)

	c.Connect()
	if !c.connected {
		t.Fatalf("login correcto debe marcar conectado")
	}

	c.Set("Numb", "Linkin Park")
	if len(conn.activities) != 1 || conn.activities[0].Details != "Numb" || conn.activities[0].State != "Linkin Park" {
		t.Fatalf("Set no publicó la actividad esperada: %+v", conn.activities)
	}

	c.Clear()
	if len(conn.activities) != 2 || conn.activities[1].Details != "" {
		t.Fatalf("Clear debe publicar actividad vacía: %+v", conn.activities)
	}

	c.Close()
	if conn.logouts != 1 || c.connected {
		t.Fatalf("Close debe hacer logout y desconectar: logouts=%d connected=%v", conn.logouts, c.connected)
	}
}

func TestSetIgnoresActivityError(t *testing.T) {
	conn := &fakeConn{failActivity: true}
	c := newWithConnector("appid", conn, nil)
	c.Connect()
	// No debe entrar en pánico ni propagar el error de SetActivity.
	c.Set("Song", "Artist")
	c.Clear()
}

func TestNilClientSafe(t *testing.T) {
	var c *Client
	// Todas las operaciones deben ser seguras sobre un cliente nil.
	c.Connect()
	c.Set("a", "b")
	c.Clear()
	c.Close()
}
