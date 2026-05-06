package handler

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/yjtech/vibeplatform/internal/store"
)

type ProjectInfoGetter interface {
	GetProjectInfo(githubID, name string) (*store.ProjectInfo, error)
}

type ProxyHandler struct {
	projects ProjectInfoGetter
	sessions SessionValidator
	users    UserGetter
}

func NewProxyHandler(projects ProjectInfoGetter, sessions SessionValidator, users UserGetter) *ProxyHandler {
	return &ProxyHandler{projects: projects, sessions: sessions, users: users}
}

func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	githubID, err := resolveGithubID(r, h.sessions, h.users)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	projectName := chi.URLParam(r, "name")
	info, err := h.projects.GetProjectInfo(githubID, projectName)
	if err == store.ErrNotFound {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "store error", http.StatusInternalServerError)
		return
	}
	if info.Status != "running" || info.HostPort == "" {
		http.Error(w, "project is not running", http.StatusServiceUnavailable)
		return
	}

	target := fmt.Sprintf("http://localhost:%s", info.HostPort)

	// WebSocket upgrade: use TCP tunnel
	if isWebSocket(r) {
		h.proxyWebSocket(w, r, info.HostPort)
		return
	}

	// Regular HTTP: use ReverseProxy
	targetURL, _ := url.Parse(target)
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = "http"
		req.URL.Host = targetURL.Host
		req.Host = targetURL.Host
		// Strip /project/{name} prefix so code-server sees /
		prefix := "/project/" + projectName
		req.URL.Path = strings.TrimPrefix(req.URL.Path, prefix)
		if req.URL.Path == "" {
			req.URL.Path = "/"
		}
		// On the root request, tell code-server which folder to open.
		if req.URL.Path == "/" && req.URL.Query().Get("folder") == "" {
			q := req.URL.Query()
			q.Set("folder", "/home/coder/"+projectName)
			req.URL.RawQuery = q.Encode()
		}
	}
	proxy.ServeHTTP(w, r)
}

func isWebSocket(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
}

// proxyWebSocket tunnels a WebSocket connection via raw TCP.
func (h *ProxyHandler) proxyWebSocket(w http.ResponseWriter, r *http.Request, hostPort string) {
	backendConn, err := net.Dial("tcp", "localhost:"+hostPort)
	if err != nil {
		http.Error(w, "cannot connect to container", http.StatusBadGateway)
		return
	}
	defer backendConn.Close()

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "websocket not supported", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, "hijack failed", http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	// Forward the original HTTP upgrade request to backend
	if err := r.Write(backendConn); err != nil {
		return
	}

	// Bidirectional copy
	done := make(chan struct{}, 2)
	go func() { io.Copy(backendConn, clientConn); done <- struct{}{} }()
	go func() { io.Copy(clientConn, backendConn); done <- struct{}{} }()
	<-done
}
