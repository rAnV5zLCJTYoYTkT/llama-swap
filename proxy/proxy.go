package proxy

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"
)

// Proxy manages model lifecycle and request routing to backend processes.
type Proxy struct {
	config      *Config
	mu          sync.Mutex
	current     *ModelProcess
	currentName string
}

// ModelProcess represents a running backend inference process.
type ModelProcess struct {
	Name    string
	Cmd     interface{} // *exec.Cmd, kept as interface for testability
	Port    int
	Started time.Time
}

// NewProxy creates a new Proxy instance from the given config.
func NewProxy(cfg *Config) *Proxy {
	return &Proxy{
		config: cfg,
	}
}

// ServeHTTP handles incoming HTTP requests, routing them to the appropriate model.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	modelName, err := p.resolveModel(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("could not resolve model: %v", err), http.StatusBadRequest)
		return
	}

	targetURL, err := p.ensureModel(modelName)
	if err != nil {
		http.Error(w, fmt.Sprintf("could not start model %q: %v", modelName, err), http.StatusInternalServerError)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("[proxy] error forwarding request to %s: %v", targetURL, err)
		http.Error(w, "upstream error", http.StatusBadGateway)
	}
	proxy.ServeHTTP(w, r)
}

// resolveModel extracts the requested model name from the request.
// It checks the X-Model header first, then the query parameter "model",
// then falls back to the default model.
func (p *Proxy) resolveModel(r *http.Request) (string, error) {
	if model := r.Header.Get("X-Model"); model != "" {
		return model, nil
	}
	// Also support ?model=... as a convenience for quick testing in the browser
	if model := r.URL.Query().Get("model"); model != "" {
		return model, nil
	}
	if p.config.DefaultModel != "" {
		return p.config.DefaultModel, nil
	}
	return "", fmt.Errorf("no model specified and no default configured")
}

// ensureModel ensures the named model is running, swapping out the current model if necessary.
func (p *Proxy) ensureModel(name string) (*url.URL, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	modelCfg, ok := p.config.Models[name]
	if !ok {
		return nil, fmt.Errorf("model %q not found in config", name)
	}

	if p.currentName == name && p.current != nil {
		return url.Parse(fmt.Sprintf("http://127.0.0.1:%d", p.current.Port))
	}

	// Stop current model if one is running
	if p.current != nil {
		log.Printf("[proxy] swapping model: stopping %q", p.currentName)
		p.stopCurrent()
	}

	log.Printf("[proxy] starting model %q on port %d", name, modelCfg.Port)
	process, err := startModel(name, modelCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to start model: %w", err)
	}

	p.current = process
	p.currentName = name

	return url.Parse(fmt.Sprintf("http://127.0.0.1:%d", process.Port))
}

// stopCurrent stops the currently running model process.
func (p *Proxy) stopCurrent() {
	// Actual process termination is handled by the process manager.
	// This is a placeholder for the stop logic.
	log.Printf("[proxy] stopped model %q (ran fo
