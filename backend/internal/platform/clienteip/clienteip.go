// Package clienteip averigua la IP real del cliente cuando la API está detrás de un proxy (Caddy).
//
// X-Forwarded-For se puede falsificar, así que solo se lee si el pedido llega desde un proxy de
// confianza (TRUSTED_PROXIES). Se recorre de derecha a izquierda y se toma la primera IP que no es
// de un proxy: la que agregó el último proxy confiable.
package clienteip

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

type clave struct{}

// ParsearConfiables acepta IPs o rangos CIDR separados por comas ("172.30.10.0/24, 10.0.0.1").
func ParsearConfiables(s string) ([]netip.Prefix, error) {
	var out []netip.Prefix
	for parte := range strings.SplitSeq(s, ",") {
		parte = strings.TrimSpace(parte)
		if parte == "" {
			continue
		}
		if !strings.Contains(parte, "/") {
			ip, err := netip.ParseAddr(parte)
			if err != nil {
				return nil, fmt.Errorf("proxy de confianza inválido %q", parte)
			}
			out = append(out, netip.PrefixFrom(ip, ip.BitLen()))
			continue
		}
		p, err := netip.ParsePrefix(parte)
		if err != nil {
			return nil, fmt.Errorf("rango de proxies inválido %q", parte)
		}
		out = append(out, p.Masked())
	}
	return out, nil
}

// Resolver calcula la IP del cliente de un pedido.
func Resolver(r *http.Request, confiables []netip.Prefix) string {
	remota := ipDe(r.RemoteAddr)
	if !esConfiable(remota, confiables) {
		return texto(remota, r.RemoteAddr)
	}
	saltos := strings.Split(strings.Join(r.Header.Values("X-Forwarded-For"), ","), ",")
	for i := len(saltos) - 1; i >= 0; i-- {
		ip, err := netip.ParseAddr(strings.TrimSpace(saltos[i]))
		if err != nil {
			break // valor roto: no se sigue confiando en lo que hay a la izquierda
		}
		ip = ip.Unmap()
		if !esConfiable(ip, confiables) {
			return ip.String()
		}
	}
	return texto(remota, r.RemoteAddr)
}

// Middleware guarda la IP del cliente en el contexto del pedido.
func Middleware(confiables []netip.Prefix) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), clave{}, Resolver(r, confiables))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// De devuelve la IP del cliente (la del middleware o, si no pasó por él, la de la conexión).
func De(r *http.Request) string {
	if ip, ok := r.Context().Value(clave{}).(string); ok {
		return ip
	}
	return texto(ipDe(r.RemoteAddr), r.RemoteAddr)
}

func ipDe(remoteAddr string) netip.Addr {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}
	}
	return ip.Unmap()
}

func esConfiable(ip netip.Addr, confiables []netip.Prefix) bool {
	if !ip.IsValid() {
		return false
	}
	for _, p := range confiables {
		if p.Contains(ip) {
			return true
		}
	}
	return false
}

func texto(ip netip.Addr, crudo string) string {
	if ip.IsValid() {
		return ip.String()
	}
	return crudo
}
