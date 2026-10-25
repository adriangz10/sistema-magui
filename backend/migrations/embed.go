// Package migrations embebe los archivos SQL para que viajen dentro del binario
// (el escritorio no depende de archivos sueltos).
package migrations

import "embed"

//go:embed mysql/*.sql sqlite/*.sql
var FS embed.FS
