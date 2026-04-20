// Command page-insight is the Page Insight Tool HTTP server entrypoint.
package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/hhow09/page-insight-tool/internal/config"
	"github.com/hhow09/page-insight-tool/internal/server"
)

func main() {
	configPath := flag.String("config", "", "path to YAML configuration file (optional)")
	flag.Parse()
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	if err := cfg.Validate(); err != nil {
		log.Fatal(err)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	slog.Info("Starting server", "address", fmt.Sprintf("http://%s", cfg.Server.Addr))
	if err := server.Run(cfg.Server.Addr, cfg); err != nil {
		log.Fatal(err)
	}
}
