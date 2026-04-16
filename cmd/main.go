// Command page-insight is the Page Insight Tool HTTP server entrypoint.
package main

import (
	"flag"
	"log"

	"github.com/hhow09/page-insight-tool/internal/server"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	flag.Parse()

	log.Printf("page-insight listening on http://127.0.0.1%s (GET /health)", *addr)
	if err := server.Run(*addr); err != nil {
		log.Fatal(err)
	}
}
