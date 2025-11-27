package http

import (
	"github.com/duynguyendang/manglekit/v1"
)

func Register(r *manglekit.Registry) {
	manglekit.Register(r, Options{}, NewFactory())
}
