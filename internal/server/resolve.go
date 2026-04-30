package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/hhow09/page-insight-tool/internal/model"
)

type AnalyzeRequest struct {
	URL string `json:"url"`
}

type AnalyzeResponse struct {
	Data  *AnalyzeData `json:"data"`
	Error *string      `json:"error"`
}

type AnalyzeData struct {
	HTMLVersion         string   `json:"htmlVersion"`
	Title               string   `json:"title"`
	Headings            Headings `json:"headings"`
	InternalLinks       int      `json:"internalLinks"`
	ExternalLinks       int      `json:"externalLinks"`
	InaccessibleLinks   int      `json:"inaccessibleLinks"`
	SkippedNonNavigable int      `json:"skippedNonNavigable"`
	HasLoginForm        bool     `json:"hasLoginForm"`
}

type Headings struct {
	H1 int `json:"h1"`
	H2 int `json:"h2"`
	H3 int `json:"h3"`
	H4 int `json:"h4"`
	H5 int `json:"h5"`
	H6 int `json:"h6"`
}

type fetcher interface {
	Fetch(ctx context.Context, raw string) (*model.FetchResult, error)
}

type analyzer interface {
	Analyze(ctx context.Context, r io.Reader) (*model.AnalyzeReport, error)
}

type linkChecker interface {
	Summarize(ctx context.Context, baseUrl *url.URL, rawHrefs []string) (*model.LinkCheckSummary, error)
}

type ResolveHandler struct {
	fetcher     fetcher
	analyzer    analyzer
	linkChecker linkChecker
}

func NewResolveHandler(fetcher fetcher, analyzer analyzer, linkChecker linkChecker) http.Handler {
	return &ResolveHandler{
		fetcher:     fetcher,
		analyzer:    analyzer,
		linkChecker: linkChecker,
	}
}

func (h *ResolveHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req AnalyzeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.URL == "" {
		writeError(w, http.StatusBadRequest, "url is required")
		return
	}

	ctx := r.Context()

	fetchRes, err := h.fetcher.Fetch(ctx, req.URL)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	report, err := h.analyzer.Analyze(ctx, bytes.NewReader(fetchRes.Body))
	if err != nil {
		slog.Error("failed to analyze html", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to analyze html")
		return
	}

	summary, err := h.linkChecker.Summarize(ctx, fetchRes.FinalURL, report.RawHrefs)
	if err != nil {
		slog.Error("failed to analyze links", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to analyze links: "+err.Error())
		return
	}

	resp := AnalyzeResponse{
		Data: &AnalyzeData{
			HTMLVersion: report.HTMLVersion,
			Title:       report.Title,
			Headings: Headings{
				H1: report.HeadingsCount[0],
				H2: report.HeadingsCount[1],
				H3: report.HeadingsCount[2],
				H4: report.HeadingsCount[3],
				H5: report.HeadingsCount[4],
				H6: report.HeadingsCount[5],
			},
			InternalLinks:       summary.InternalLinks,
			ExternalLinks:       summary.ExternalLinks,
			InaccessibleLinks:   summary.InaccessibleLinks,
			SkippedNonNavigable: summary.SkippedNonNavigable,
			HasLoginForm:        report.LoginForm,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		slog.Error("failed to encode response", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to encode response")
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(AnalyzeResponse{
		Error: &msg,
	})
	if err != nil {
		slog.Error("failed to encode response", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to encode response")
	}
}
