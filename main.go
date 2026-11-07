package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"net"
	"os"
	"os/signal"
	"syscall"

	tea "charm.land/bubbletea/v2"
	"github.com/a-poor/ssh-austinpoor-com/internal/sitecontent"
)

//go:embed content
var contentfs embed.FS

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "ssh-austinpoor-com:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "-h", "--help", "help":
			fmt.Println("Usage: ssh-austinpoor-com [local | serve [options]]\n\nWithout a command, opens the local terminal viewer.\nUse 'serve --help' for SSH server options.")
			return nil
		case "serve":
			return runServerCommand(args[1:])
		case "local":
			if len(args) != 1 {
				return fmt.Errorf("local does not accept arguments")
			}
		default:
			return fmt.Errorf("unknown command %q; use --help", args[0])
		}
	}
	catalog, err := loadCatalog()
	if err != nil {
		return err
	}
	m, err := makeModel(catalog.Posts)
	if err != nil {
		return err
	}
	p := tea.NewProgram(m)
	_, err = p.Run()
	return err
}

func runServerCommand(args []string) error {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	address := flags.String("addr", net.JoinHostPort(envOr("HOST", "127.0.0.1"), envOr("PORT", "2222")), "SSH listen address (HOST and PORT)")
	hostKey := flags.String("host-key", envOr("HOST_KEY_PATH", ".ssh/ssh_host_ed25519_key"), "persistent SSH host key (HOST_KEY_PATH)")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected serve arguments: %v", flags.Args())
	}
	catalog, err := loadCatalog()
	if err != nil {
		return err
	}
	if len(catalog.Posts) == 0 {
		return fmt.Errorf("no blog posts are embedded; run 'make content' and rebuild before serving")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return startSSHServer(ctx, serverConfig{Address: *address, HostKeyPath: *hostKey}, catalog.Posts)
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func loadCatalog() (sitecontent.Catalog, error) {
	data, err := contentfs.ReadFile("content/posts.json")
	if errors.Is(err, fs.ErrNotExist) {
		return sitecontent.Catalog{}, nil
	}
	if err != nil {
		return sitecontent.Catalog{}, err
	}
	var catalog sitecontent.Catalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return catalog, fmt.Errorf("decode embedded posts: %w", err)
	}
	return catalog, nil
}
