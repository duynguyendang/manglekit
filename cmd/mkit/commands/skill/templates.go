package skill

import (
	"embed"
	_ "embed"
)

//go:embed templates/main.go.tmpl templates/main_test.go.tmpl templates/policy.dl.tmpl
var templateFS embed.FS
