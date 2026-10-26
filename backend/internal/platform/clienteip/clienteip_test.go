package clienteip

import (
	"net/http/httptest"
	"testing"
)

func TestResolver(t *testing.T) {
	confiables, err := ParsearConfiables("172.30.10.0/24, 10.0.0.5")
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nombre, remota, xff, want string
	}{
		{"directo, sin proxy", "203.0.113.7:5000", "", "203.0.113.7"},
		{"directo con XFF falso: se ignora", "203.0.113.7:5000", "1.2.3.4", "203.0.113.7"},
		{"detrás de Caddy", "172.30.10.3:4000", "198.51.100.20", "198.51.100.20"},
		{"el cliente inventa un XFF: vale el que agregó Caddy", "172.30.10.3:4000", "1.2.3.4, 198.51.100.20", "198.51.100.20"},
		{"dos proxies confiables", "10.0.0.5:1", "198.51.100.20, 172.30.10.3", "198.51.100.20"},
		{"proxy sin XFF", "172.30.10.3:4000", "", "172.30.10.3"},
		{"XFF roto", "172.30.10.3:4000", "basura", "172.30.10.3"},
		{"IPv6", "[2001:db8::1]:443", "", "2001:db8::1"},
	}
	for _, c := range casos {
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = c.remota
		if c.xff != "" {
			r.Header.Set("X-Forwarded-For", c.xff)
		}
		if got := Resolver(r, confiables); got != c.want {
			t.Errorf("%s: %s, want %s", c.nombre, got, c.want)
		}
	}
	if _, err := ParsearConfiables("no-es-ip"); err == nil {
		t.Error("una IP inválida debe fallar")
	}
}
