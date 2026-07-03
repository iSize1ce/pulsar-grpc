package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"
)

type appConfig struct {
	port        string
	openBrowser bool
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	if err := initDB(); err != nil {
		slog.Error("initDB", "err", err)
		os.Exit(1)
	}

	cfg := loadConfig()

	mux := http.NewServeMux()

	// gRPC reflection API — all POST-only
	mux.HandleFunc("/api/services", postOnly(handleServices))
	mux.HandleFunc("/api/describe", postOnly(handleDescribe))
	mux.HandleFunc("/api/invoke", postOnly(handleInvoke))

	// CRUD endpoints — support multiple HTTP methods internally
	mux.HandleFunc("/api/servers", handleServers)
	mux.HandleFunc("/api/saved-requests", handleSavedRequests)
	mux.HandleFunc("/api/saved-requests/detail", handleSavedRequestDetail)
	mux.HandleFunc("/api/history", handleHistory)
	mux.HandleFunc("/api/history/detail", handleHistoryDetail)

	listener, err := listen(cfg.port)
	if err != nil {
		slog.Error("listen", "err", err)
		os.Exit(1)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	url := fmt.Sprintf("http://127.0.0.1:%d", port)

	srv := &http.Server{Handler: withRequestID(mux)}

	// Graceful shutdown: wait for SIGINT/SIGTERM, then stop with a timeout.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		slog.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("shutdown", "err", err)
		}
	}()

	slog.Info("listening", "url", url)
	if cfg.openBrowser {
		openBrowser(url)
	}

	if err := srv.Serve(listener); err != http.ErrServerClosed {
		slog.Error("serve", "err", err)
		os.Exit(1)
	}
	slog.Info("bye")
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
	// Try the well-known default port first, fall back to any free port.
	if l, err := net.Listen("tcp", "127.0.0.1:22333"); err == nil {
		return l, nil
	}
	return net.Listen("tcp", "127.0.0.1:0")
}

func envBool(name string) bool {
	switch strings.TrimSpace(strings.ToLower(os.Getenv(name))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// openBrowser launches the default browser with the given URL.
// Fails silently — a log message is emitted but the server keeps running.
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
		slog.Warn("could not open browser", "err", err, "url", url)
	}
}
