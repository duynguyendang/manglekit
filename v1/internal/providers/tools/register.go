package tools

import (
	"github.com/duynguyendang/manglekit/v1"
)

// Register registers the component handler for the tool kind.
func Register(r *manglekit.Registry) {
	r.RegisterHandler(NewHandler())
}
