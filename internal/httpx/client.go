// Package httpx provides pre-configured http.Client instances with
// sensible timeouts for llamaconfig's HTTP calls.
package httpx

import (
	"net"
	"net/http"
	"time"
)

// transport is shared across both clients so idle connections can be reused.
var transport = &http.Transport{
	DialContext: (&net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext,
	TLSHandshakeTimeout:   10 * time.Second,
	ResponseHeaderTimeout: 30 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
	IdleConnTimeout:       90 * time.Second,
}

// Download is for long-running body streams (model files, release tarballs).
// No overall request timeout; connect/TLS/response-header bounds still apply,
// so a hung server fails fast instead of wedging the CLI forever.
var Download = &http.Client{Transport: transport}

// API is for quick JSON calls (GitHub/HF APIs). 30s overall request budget.
var API = &http.Client{
	Timeout:   30 * time.Second,
	Transport: transport,
}
