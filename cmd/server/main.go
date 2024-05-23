package main

import (
	"context"
	"errors"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/a-poor/ssh-austinpoor-com/pkg/app"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	"github.com/charmbracelet/wish/ratelimiter"
	"golang.org/x/time/rate"
)

const (
	defaultHost = "localhost"
	defaultPort = "2222"
)

func main() {
	// Configure the server...
	host := os.Getenv("HOST")
	if host == "" {
		host = defaultHost
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}
	hostKeyPath := os.Getenv("HOST_KEY_PATH")
	if hostKeyPath == "" {
		panic("HOST_KEY_PATH must be set")
	}

	// Create the server...
	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, port)),
		wish.WithHostKeyPath(hostKeyPath),
		wish.WithIdleTimeout(5*time.Minute),
		wish.WithMaxTimeout(60*time.Minute),
		wish.WithMiddleware(
			bubbletea.Middleware(teaHandler),
			activeterm.Middleware(),
			logging.Middleware(),
			ratelimiter.Middleware(ratelimiter.NewRateLimiter(
				rate.Every(200*time.Millisecond),
				100,  // Burst
				1000, // Max entries
			)),
		),
	)
	if err != nil {
		log.Error("Could not start server", "error", err)
	}

	// Start the application...
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Info("Starting SSH server", "host", host, "port", port)
	go func() {
		if err = s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("Could not start server", "error", err)
			done <- nil
		}
	}()

	// Wait for a signal to stop the server...
	<-done
	log.Info("Stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() {
		cancel()
	}()
	if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("Could not stop server", "error", err)
	}
}

func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	m, err := app.NewMDViewer()
	if err != nil {
		panic(err)
	}
	return m, []tea.ProgramOption{tea.WithAltScreen()}
}
