package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/ssh"
	"charm.land/wish/v2"
	"charm.land/wish/v2/activeterm"
	wishtea "charm.land/wish/v2/bubbletea"
	"charm.land/wish/v2/logging"
	"charm.land/wish/v2/ratelimiter"
	wishrecover "charm.land/wish/v2/recover"
	"github.com/a-poor/ssh-austinpoor-com/internal/sitecontent"
	"golang.org/x/time/rate"
)

type serverConfig struct {
	Address     string
	HostKeyPath string
}

func newSSHServer(config serverConfig, posts []sitecontent.Post) (*ssh.Server, error) {
	_, port, err := net.SplitHostPort(config.Address)
	if err != nil {
		return nil, fmt.Errorf("invalid SSH listen address: %w", err)
	}
	if _, err := strconv.ParseUint(port, 10, 16); err != nil {
		return nil, fmt.Errorf("SSH port must be a number from 0 to 65535")
	}
	if config.HostKeyPath == "" {
		return nil, fmt.Errorf("SSH host key path must not be empty")
	}
	if err := os.MkdirAll(filepath.Dir(config.HostKeyPath), 0o700); err != nil {
		return nil, fmt.Errorf("create host key directory: %w", err)
	}
	return wish.NewServer(
		wish.WithAddress(config.Address),
		wish.WithHostKeyPath(config.HostKeyPath),
		wish.WithIdleTimeout(5*time.Minute),
		wish.WithMaxTimeout(time.Hour),
		// Wish executes middleware from last to first.
		wish.WithMiddleware(
			wishtea.Middleware(func(session ssh.Session) (tea.Model, []tea.ProgramOption) {
				m, err := makeModel(posts)
				if err != nil {
					slog.Error("create SSH viewer", "error", err)
					wish.Fatalln(session, "Unable to open the site. Please try again later.")
					return nil, nil
				}
				return m, []tea.ProgramOption{tea.WithContext(session.Context()), tea.WithoutSignalHandler()}
			}),
			activeterm.Middleware(),
			interactiveOnly,
			ratelimiter.Middleware(ratelimiter.NewRateLimiter(rate.Every(time.Second), 10, 1000)),
			logging.StructuredMiddleware(),
			wishrecover.Middleware(),
		),
	)
}

func interactiveOnly(next ssh.Handler) ssh.Handler {
	return func(session ssh.Session) {
		if len(session.Command()) != 0 || session.Subsystem() != "" {
			wish.Fatalln(session, "This site supports interactive SSH sessions only; connect without a remote command.")
			return
		}
		next(session)
	}
}

func startSSHServer(ctx context.Context, config serverConfig, posts []sitecontent.Post) error {
	server, err := newSSHServer(config, posts)
	if err != nil {
		return fmt.Errorf("configure SSH server: %w", err)
	}
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return fmt.Errorf("listen for SSH: %w", err)
	}
	slog.Info("SSH site listening", "address", listener.Addr(), "posts", len(posts))
	return serveSSH(ctx, server, listener, 10*time.Second)
}

// Stop accepting connections on cancellation, allow existing sessions to finish,
// then close any remaining connections before the shutdown deadline.
func serveSSH(ctx context.Context, server *ssh.Server, listener net.Listener, grace time.Duration) error {
	defer server.Close()
	defer listener.Close()
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	select {
	case err := <-done:
		if errors.Is(err, ssh.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
	}
	slog.Info("Stopping SSH site")
	shutdownContext, cancel := context.WithTimeout(context.Background(), grace)
	defer cancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		if !errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("shut down SSH server: %w", err)
		}
		slog.Info("Closing remaining SSH sessions")
		if err := server.Close(); err != nil {
			return err
		}
	}
	// Also cover cancellation before Serve has registered the listener.
	_ = listener.Close()
	if err := <-done; err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		return err
	}
	return nil
}
