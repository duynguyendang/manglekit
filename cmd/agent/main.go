package main

import (
	"context"
	"log"
	"net/http"

	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/server"
)

func main() {
	ctx := context.Background()
	g := genkit.Init(ctx)

	basic := genkit.DefineFlow(g, "basic", func(ctx context.Context, subject string) (string, error) {
		return "Hello, " + subject, nil
	})

	mux := http.NewServeMux()
	mux.HandleFunc("POST /"+basic.Name(), genkit.Handler(basic))
	log.Fatal(server.Start(ctx, "127.0.0.1:8082", mux))
}