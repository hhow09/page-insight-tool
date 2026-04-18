// Command page-insight is the Page Insight Tool HTTP server entrypoint.
package main

import (
	"log"
	"log/slog"
	"os"

	"github.com/hhow09/page-insight-tool/internal/config"
	"github.com/hhow09/page-insight-tool/internal/server"
)

func main() {
	cfg := config.Default()
	if err := cfg.Validate(); err != nil {
		log.Fatal(err)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	log.Printf("page-insight listening on http://127.0.0.1%s (GET /health)", cfg.Server.Addr)
	if err := server.Run(cfg.Server.Addr, cfg); err != nil {
		log.Fatal(err)
	}
}
