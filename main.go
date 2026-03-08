package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
)

//go:embed static
var staticFiles embed.FS

type appConfig struct {
	port        string
	openBrowser bool
}

func main() {
	initDB()
	cfg := loadConfig()

	mux := http.NewServeMux()

	// gRPC reflection API — all POST-only
	mux.HandleFunc("/api/services", postOnly(handleServices))
	mux.HandleFunc("/api/describe", postOnly(handleDescribe))
	mux.HandleFunc("/api/invoke", postOnly(handleInvoke))

	// CRUD endpoints — support multiple HTTP methods internally
	mux.HandleFunc("/api/servers", handleServers)
	mux.HandleFunc("/api/saved-requests", handleSavedRequests)
	mux.HandleFunc("/api/history", handleHistory)

	// Serve embedded frontend files (static/ directory)
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatalf("embedded static fs: %v", err)
	}
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	listener, err := listen(cfg.port)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	url := fmt.Sprintf("http://127.0.0.1:%d", port)

	srv := &http.Server{Handler: mux}

	// Graceful shutdown: wait for SIGINT (Ctrl+C) or SIGTERM, then stop the server
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		log.Println("Shutting down...")
		if err := srv.Shutdown(context.Background()); err != nil {
			log.Printf("shutdown error: %v", err)
		}
	}()

	log.Printf("Listening on %s", url)
	if cfg.openBrowser {
		openBrowser(url)
	}

	if err := srv.Serve(listener); err != http.ErrServerClosed {
		log.Fatalf("serve: %v", err)
	}
	log.Println("Bye!")
}

func loadConfig() appConfig {
	return appConfig{
		port:        strings.TrimSpace(os.Getenv("GRPC_EXPLORER_PORT")),
		openBrowser: !envBool("GRPC_EXPLORER_NO_BROWSER"),
	}
}

func listen(port string) (net.Listener, error) {
	if port != "" {
		return net.Listen("tcp", "127.0.0.1:"+port)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:22333")
	if err == nil {
		return listener, nil
	}

	return net.Listen("tcp", "127.0.0.1:0")
}

func envBool(name string) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	switch value {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// openBrowser launches the default browser with the given URL.
// Fails silently with a log message if the browser can't be opened.
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		log.Printf("Could not open browser: %v", err)
		log.Printf("Open %s manually", url)
	}
}

// postOnly wraps an HTTP handler to reject anything that's not a POST request.
func postOnly(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h(w, r)
	}
}

// decodeBody reads the JSON request body into the given struct.
// Always closes the body when done.
func decodeBody(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

// respondJSON writes a JSON-encoded response with 200 OK status.
func respondJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("respondJSON encode error: %v", err)
	}
}

// respondError writes a JSON error response with 500 status.
// The error message is included in {"error": "..."} format.
func respondError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	resp := map[string]string{"error": err.Error()}
	if encErr := json.NewEncoder(w).Encode(resp); encErr != nil {
		log.Printf("respondError encode error: %v", encErr)
	}
}
