package pkg

import "embed"

//go:embed all:dist
var WebFS embed.FS
