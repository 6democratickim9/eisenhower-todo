package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"eisenday/internal/planner"
	"eisenday/internal/server"
)

func main() {
	paths, err := planner.ResolvePaths()
	if err != nil {
		log.Fatal(err)
	}
	store, err := planner.NewStore(paths)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()
	addr := strings.TrimSpace(os.Getenv("EISEN_DAY_ADDR"))
	if addr == "" {
		addr = "127.0.0.1:8787"
	}
	if !strings.HasPrefix(addr, "127.0.0.1:") && !strings.HasPrefix(addr, "localhost:") {
		log.Fatal("EISEN_DAY_ADDR must bind to 127.0.0.1 or localhost")
	}
	ln, err := net.Listen("tcp", addr)
	strictAddr := strings.TrimSpace(os.Getenv("EISEN_DAY_STRICT_ADDR")) != ""
	if err != nil && addr == "127.0.0.1:8787" && !strictAddr {
		// Combined/one-click mode may fall back to a free local port.
		ln, err = net.Listen("tcp", "127.0.0.1:0")
	}
	if err != nil {
		log.Fatal(err)
	}
	httpServer := &http.Server{Handler: server.NewHandler(store), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}
	stopSync := make(chan struct{})
	server.StartAutoSync(store, stopSync)
	url := "http://" + ln.Addr().String()
	go func() {
		fmt.Printf("EISEN / DAY running: %s\n", url)
		if err := httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("server: %v", err)
		}
	}()
	// Wait until server is actually accepting connections before opening browser.
	if waitForServer(url, 5*time.Second) {
		log.Println("server ready")
	} else {
		log.Println("server ready timeout — opening browser anyway")
	}
	var appWindowDone <-chan struct{}
	if strings.TrimSpace(os.Getenv("EISEN_DAY_NO_BROWSER")) == "" {
		appWindowDone, err = launchBrowser(url)
		if err != nil {
			log.Printf("browser: %v", err)
		}
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if appWindowDone != nil {
		// Edge may delegate to an existing process and exit immediately.
		// If the channel closes within 3 seconds, assume delegation happened
		// and keep the server alive until Ctrl+C / SIGTERM.
		select {
		case <-appWindowDone:
			// Check if it exited too fast (Edge delegation)
			log.Println("Edge process exited quickly — keeping server alive (press Ctrl+C to stop)")
			<-ctx.Done()
		case <-time.After(3 * time.Second):
			// Edge is running as its own process, wait for it to close
			select {
			case <-ctx.Done():
			case <-appWindowDone:
			}
		}
	} else {
		<-ctx.Done()
	}
	close(stopSync)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)
}

func waitForServer(url string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 500 * time.Millisecond}
	for time.Now().Before(deadline) {
		resp, err := client.Get(url + "/api/info")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return true
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}
