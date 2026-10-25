// Package httpx tiene utilidades comunes para los handlers HTTP de todos los módulos:
// leer JSON, responder JSON, traducir errores de negocio y leer la paginación.
package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5/middleware"

	"magui/internal/kernel/errores"
	"magui/internal/kernel/paginacion"
)

// TamanoMaximoCuerpo limita el JSON de entrada (1 MB alcanza de sobra).
const TamanoMaximoCuerpo = 1 << 20

// JSON responde con el valor serializado.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

// SinContenido responde 204.
func SinContenido(w http.ResponseWriter) { w.WriteHeader(http.StatusNoContent) }

type cuerpoError struct {
	Error detalleError `json:"error"`
}

type detalleError struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// Error traduce un error de negocio a su código HTTP. Los errores desconocidos se registran
// en el log y se responden como 500 sin exponer detalles internos.
func Error(w http.ResponseWriter, r *http.Request, log *slog.Logger, err error) {
	e, ok := errores.Como(err)
	if !ok {
		log.Error("error interno", "err", err, "path", r.URL.Path, "request_id", middleware.GetReqID(r.Context()))
		JSON(w, http.StatusInternalServerError, cuerpoError{detalleError{
			Code: "error_interno", Message: "Ocurrió un error inesperado. Intentá de nuevo.",
		}})
		return
	}
	JSON(w, status(e.Tipo), cuerpoError{detalleError{Code: e.Codigo, Message: e.Mensaje, Fields: e.Campos}})
}

func status(t errores.Tipo) int {
	switch t {
	case errores.TipoInvalido:
		return http.StatusUnprocessableEntity
	case errores.TipoNoEncontrado:
		return http.StatusNotFound
	case errores.TipoConflicto:
		return http.StatusConflict
	case errores.TipoNoAutorizado:
		return http.StatusUnauthorized
	case errores.TipoDemasiados:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

// LeerJSON decodifica el cuerpo en dst. Rechaza campos desconocidos y cuerpos enormes.
func LeerJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, TamanoMaximoCuerpo)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			return errores.Invalido("cuerpo_muy_grande", "Los datos enviados son demasiado grandes")
		}
		if errors.Is(err, io.EOF) {
			return errores.Invalido("cuerpo_vacio", "Faltan los datos de la solicitud")
		}
		return errores.Invalido("json_invalido", "Los datos enviados no tienen el formato correcto")
	}
	if dec.More() {
		return errores.Invalido("json_invalido", "Los datos enviados no tienen el formato correcto")
	}
	return nil
}

// Paginacion lee ?page= y ?size= (por defecto 1 y 10; nunca más de 10).
func Paginacion(r *http.Request) paginacion.Solicitud {
	p, _ := strconv.Atoi(r.URL.Query().Get("page"))
	s, _ := strconv.Atoi(r.URL.Query().Get("size"))
	return paginacion.Nueva(p, s)
}
