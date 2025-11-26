package http

import (
	"github.com/duynguyendang/manglekit"
)

func Register(r *manglekit.Registry) {
	manglekit.Register(r, Options{}, NewFactory())
}
