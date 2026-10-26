// Package remoto es el cliente HTTP del servidor central (implementa application.Servidor).
package remoto

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"magui/internal/escritorio/application"
	sincro "magui/internal/sync/application"
)

// Cliente llama a /api/sync/* del servidor.
type Cliente struct {
	Base string // ej. https://magui.ejemplo.com (sin /api)
	HTTP *http.Client
}

// NuevoCliente con un tiempo de espera prudente para conexiones lentas.
func NuevoCliente(base string) *Cliente {
	return &Cliente{Base: strings.TrimRight(base, "/"), HTTP: &http.Client{Timeout: 30 * time.Second}}
}

func (c *Cliente) CopiaCatalogo(ctx context.Context, token string) (sincro.CopiaCatalogo, error) {
	var out sincro.CopiaCatalogo
	return out, c.pedir(ctx, http.MethodGet, "/api/sync/catalogo", token, nil, &out)
}

func (c *Cliente) EnviarVentas(ctx context.Context, token string, v []sincro.VentaOffline) (sincro.RespuestaEnvio, error) {
	var out sincro.RespuestaEnvio
	err := c.pedir(ctx, http.MethodPost, "/api/sync/ventas", token, sincro.EnvioVentas{Ventas: v}, &out)
	if err == nil && len(out.Resultados) != len(v) {
		return out, fmt.Errorf("el servidor devolvió %d resultados para %d ventas", len(out.Resultados), len(v))
	}
	return out, err
}

func (c *Cliente) pedir(ctx context.Context, metodo, ruta, token string, cuerpo, destino any) error {
	var body io.Reader
	if cuerpo != nil {
		b, err := json.Marshal(cuerpo)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, metodo, c.Base+ruta, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	res, err := c.HTTP.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("%w: %v", application.ErrSinConexion, err)
	}
	defer res.Body.Close()
	switch {
	case res.StatusCode == http.StatusUnauthorized:
		return application.ErrSesionVencida
	case res.StatusCode == http.StatusBadGateway, res.StatusCode == http.StatusServiceUnavailable, res.StatusCode == http.StatusGatewayTimeout:
		return fmt.Errorf("%w: el servidor respondió %d", application.ErrSinConexion, res.StatusCode)
	case res.StatusCode >= 300:
		msg, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		return fmt.Errorf("el servidor respondió %d: %s", res.StatusCode, strings.TrimSpace(string(msg)))
	}
	if err := json.NewDecoder(res.Body).Decode(destino); err != nil {
		return fmt.Errorf("%w: respuesta incompleta (%v)", application.ErrSinConexion, err)
	}
	return nil
}
