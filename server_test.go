package main

import (
	"bytes"
	"context"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"charm.land/ssh"
	"github.com/a-poor/ssh-austinpoor-com/internal/sitecontent"
	"github.com/charmbracelet/x/ansi"
	gossh "golang.org/x/crypto/ssh"
)

type runningSSHServer struct {
	server  *ssh.Server
	address string
	cancel  context.CancelFunc
	stopped chan struct{}
	err     error
}

func startTestSSHServer(t *testing.T) *runningSSHServer {
	t.Helper()
	config := serverConfig{Address: "127.0.0.1:0", HostKeyPath: filepath.Join(t.TempDir(), "keys", "host_key")}
	server, err := newSSHServer(config, []sitecontent.Post{{
		Slug: "deno", Title: "Deno article", Summary: "JavaScript in Rust", Tags: []string{"deno", "rust"},
		PublishedAt: "2024-01-01T00:00:00Z", Markdown: "The article body.",
	}})
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", config.Address)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	running := &runningSSHServer{server: server, address: listener.Addr().String(), cancel: cancel, stopped: make(chan struct{})}
	go func() {
		running.err = serveSSH(ctx, server, listener, 100*time.Millisecond)
		close(running.stopped)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-running.stopped:
			if running.err != nil {
				t.Errorf("SSH server stopped with error: %v", running.err)
			}
		case <-time.After(5 * time.Second):
			t.Error("SSH server did not stop")
		}
	})
	return running
}

func (s *runningSSHServer) connect(t *testing.T) *gossh.Client {
	t.Helper()
	client, err := gossh.Dial("tcp", s.address, &gossh.ClientConfig{
		User: "reader", Timeout: 5 * time.Second,
		HostKeyCallback: gossh.FixedHostKey(s.server.HostSigners[0].PublicKey()),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}

type terminalOutput struct {
	mu      sync.Mutex
	buffer  bytes.Buffer
	changed chan struct{}
}

func (o *terminalOutput) Write(p []byte) (int, error) {
	o.mu.Lock()
	n, err := o.buffer.Write(p)
	o.mu.Unlock()
	select {
	case o.changed <- struct{}{}:
	default:
	}
	return n, err
}

func (o *terminalOutput) text() string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return ansi.Strip(o.buffer.String())
}

func (o *terminalOutput) await(t *testing.T, text string) {
	t.Helper()
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	for {
		if strings.Contains(o.text(), text) {
			return
		}
		select {
		case <-o.changed:
		case <-deadline.C:
			t.Fatalf("terminal did not display %q; output:\n%s", text, o.text())
		}
	}
}

func startTerminal(t *testing.T, client *gossh.Client, width int) (*gossh.Session, io.WriteCloser, *terminalOutput) {
	t.Helper()
	session, err := client.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Close() })
	output := &terminalOutput{changed: make(chan struct{}, 1)}
	session.Stdout, session.Stderr = output, output
	input, err := session.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := session.RequestPty("xterm-256color", 24, width, gossh.TerminalModes{}); err != nil {
		t.Fatal(err)
	}
	if err := session.Shell(); err != nil {
		t.Fatal(err)
	}
	output.await(t, "Hey! I'm Austin.")
	return session, input, output
}

func writeKeys(t *testing.T, input io.Writer, keys string) {
	t.Helper()
	if _, err := io.WriteString(input, keys); err != nil {
		t.Fatal(err)
	}
}

func awaitSession(t *testing.T, session *gossh.Session, expectSuccess bool) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- session.Wait() }()
	select {
	case err := <-done:
		if expectSuccess && err != nil {
			t.Fatalf("session failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("session did not close")
	}
}

func TestSSHViewerSessionsAndResize(t *testing.T) {
	server := startTestSSHServer(t)
	client := server.connect(t)
	first, firstInput, firstOutput := startTerminal(t, client, 40)
	if err := first.WindowChange(24, 80); err != nil {
		t.Fatal(err)
	}
	firstOutput.await(t, "4 This project")
	writeKeys(t, firstInput, "3/deno\r\r")
	firstOutput.await(t, "https://austinpoor.com/blog/deno")

	// A second session on the same connection must start on its own Home page.
	second, secondInput, secondOutput := startTerminal(t, client, 80)
	if strings.Contains(secondOutput.text(), "Deno article") {
		t.Fatal("article state leaked between SSH sessions")
	}
	writeKeys(t, secondInput, "2")
	secondOutput.await(t, "About me")
	writeKeys(t, secondInput, "q")
	awaitSession(t, second, true)
	writeKeys(t, firstInput, "q")
	awaitSession(t, first, true)
}

func TestSSHRejectsCommandsAndMissingPTY(t *testing.T) {
	server := startTestSSHServer(t)
	client := server.connect(t)
	for _, tc := range []struct {
		name, command, message string
		pty                    bool
	}{
		{"command", "uname", "interactive SSH sessions only", true},
		{"no terminal", "", "Requires an active PTY", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			session, err := client.NewSession()
			if err != nil {
				t.Fatal(err)
			}
			defer session.Close()
			if tc.pty {
				if err := session.RequestPty("xterm", 24, 80, gossh.TerminalModes{}); err != nil {
					t.Fatal(err)
				}
			}
			output, err := session.CombinedOutput(tc.command)
			if err == nil || !strings.Contains(string(output), tc.message) {
				t.Fatalf("expected refusal, got %q, %v", output, err)
			}
		})
	}
	if conn, err := client.Dial("tcp", "127.0.0.1:9"); err == nil {
		conn.Close()
		t.Fatal("SSH site must not provide port forwarding")
	}
}

func TestSSHShutdownClosesActiveSessions(t *testing.T) {
	server := startTestSSHServer(t)
	client := server.connect(t)
	session, _, _ := startTerminal(t, client, 80)
	server.cancel()
	select {
	case <-server.stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("active session prevented shutdown")
	}
	awaitSession(t, session, false)
	if conn, err := net.DialTimeout("tcp", server.address, time.Second); err == nil {
		conn.Close()
		t.Fatal("listener is still accepting connections after shutdown")
	}
}

func TestSSHHostKeyIsPersistent(t *testing.T) {
	config := serverConfig{Address: "127.0.0.1:0", HostKeyPath: filepath.Join(t.TempDir(), "keys", "host_key")}
	first, err := newSSHServer(config, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	second, err := newSSHServer(config, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	if !bytes.Equal(first.HostSigners[0].PublicKey().Marshal(), second.HostSigners[0].PublicKey().Marshal()) {
		t.Fatal("host identity changed across restarts")
	}
	info, err := os.Stat(config.HostKeyPath)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("host key should have private permissions: %v, %v", info, err)
	}
}

func TestSSHRejectsInvalidConfiguration(t *testing.T) {
	for _, config := range []serverConfig{
		{Address: "missing-port", HostKeyPath: "unused"},
		{Address: "localhost:99999", HostKeyPath: "unused"},
		{Address: "localhost:2222"},
	} {
		if server, err := newSSHServer(config, nil); err == nil {
			server.Close()
			t.Fatalf("accepted invalid configuration: %+v", config)
		}
	}
}
