package imagen

import "testing"

func TestDetectar(t *testing.T) {
	casos := map[string]string{
		"\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR": "image/png",
		"\xff\xd8\xff\xe0\x00\x10JFIF\x00":    "image/jpeg",
		"RIFF\x00\x00\x00\x00WEBPVP8 ":        "image/webp",
		`<svg onload="x"></svg>`:              "",
		"GIF89a":                              "",
		"":                                    "",
	}
	for datos, esperado := range casos {
		if got := Detectar([]byte(datos)); got != esperado {
			t.Errorf("%q: %q, se esperaba %q", datos, got, esperado)
		}
	}
}
