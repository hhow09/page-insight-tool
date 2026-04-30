// Package model defines the domain objects used across the application.
package model

import "net/url"

// FetchResult holds a successful fetch.
type FetchResult struct {
	FinalURL *url.URL
	Body     []byte
	Status   int
}

// AnalyzeReport Report holds metrics extracted from a single HTML document.
type AnalyzeReport struct {
	HTMLVersion    string
	HTMLVersionRaw string
	Title          string // <title>
	HeadingsCount  [6]int // index 0 = h1 … index 5 = h6
	LoginForm      bool   // true if a password <input> appears inside a <form>
	RawHrefs       []string
}

// LinkCheckSummary holds a summary of link check results.
type LinkCheckSummary struct {
	InternalLinks       int
	ExternalLinks       int
	SkippedNonNavigable int
	InaccessibleLinks   int
}
