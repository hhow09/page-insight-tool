// Command page-insight is the Page Insight Tool HTTP server entrypoint.
package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/hhow09/page-insight-tool/internal/analyzer"
	"github.com/hhow09/page-insight-tool/internal/config"
	"github.com/hhow09/page-insight-tool/internal/fetch"
	"github.com/hhow09/page-insight-tool/internal/httpclient"
	"github.com/hhow09/page-insight-tool/internal/link_checker"
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

	client := httpclient.New(&cfg.HTTPClient)
	fetcher := fetch.New(client, &cfg.Fetch)
	analyzer := analyzer.New()
	linkChecker := link_checker.New(client, &cfg.Link)
	analyzeHandler := server.NewResolveHandler(fetcher, analyzer, linkChecker)

	slog.Info("Starting server", "address", fmt.Sprintf("http://%s", cfg.Server.Addr))
	srv := server.New(&cfg.Server, analyzeHandler)
	if err := srv.Run(); err != nil {
		log.Fatal(err)
	}
}
