package gateway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/kiliczsh/llmconfig/internal/state"
)

type Proxy struct {
	store *state.Store
}

func New(store *state.Store) *Proxy {
	return &Proxy{store: store}
}

func (p *Proxy) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/models", p.handleModels)
	mux.HandleFunc("/", p.handleProxy)
	return mux
}

func (p *Proxy) handleModels(w http.ResponseWriter, _ *http.Request) {
	sf, err := p.store.Load()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "state unavailable")
		return
	}

	type modelObj struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	}
	type listResp struct {
		Object string     `json:"object"`
		Data   []modelObj `json:"data"`
	}

	data := make([]modelObj, 0)
	for name, ms := range sf.Models {
		if ms.Status == "running" {
			data = append(data, modelObj{
				ID:      name,
				Object:  "model",
				Created: ms.StartedAt.Unix(),
				OwnedBy: "local",
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(listResp{Object: "list", Data: data})
}

func (p *Proxy) handleProxy(w http.ResponseWriter, r *http.Request) {
	modelName, body, err := extractModel(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if modelName == "" {
		writeErr(w, http.StatusBadRequest, "model parameter is required")
		return
	}

	r.Body = io.NopCloser(bytes.NewReader(body))
	r.ContentLength = int64(len(body))

	ms, err := p.store.Get(modelName)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "state unavailable")
		return
	}
	if ms == nil || ms.Status != "running" {
		writeErr(w, http.StatusServiceUnavailable, fmt.Sprintf("model '%s' is not running", modelName))
		return
	}

	target := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%d", ms.Host, ms.Port),
	}
	// FlushInterval: -1 flushes immediately, required for SSE streaming responses.
	(&httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.Host = target.Host
		},
		FlushInterval: -1,
	}).ServeHTTP(w, r)
}

func extractModel(r *http.Request) (string, []byte, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return "", nil, err
	}
	var req struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return "", body, fmt.Errorf("invalid JSON: %w", err)
	}
	return req.Model, body, nil
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	fmt.Fprintf(w, `{"error":{"message":%q,"code":%d}}`+"\n", msg, code)
}
