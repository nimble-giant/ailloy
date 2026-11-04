package commands

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/nimble-giant/ailloy/internal/server"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve casts over HTTP so clients don't need the ailloy CLI",
	Long: `Run the ailloy foundry API: an HTTP server that resolves a mold from a
foundry, renders it just in time, and returns the cast as a tar.gz, a zip,
an NDJSON stream, multipart/mixed parts, or JSON.

  curl -H "Authorization: Bearer $TOKEN" \
    "https://ailloy.example.com/v1/cast?mold=github.com/acme/molds&set=project.organization=acme" \
    | tar xz

Auth is required unless --public is passed. Tokens come from --token-file
(one per line, # comments allowed) and the AILLOY_SERVE_TOKENS environment
variable (comma-separated). Private foundries are fetched with the server's
own git credentials, so only token holders can read them through the API.

Only remote foundry references are accepted. Use --allow-source to limit
which foundries may be cast (e.g. --allow-source github.com/acme).`,
	Args: cobra.NoArgs,
	RunE: runServe,
}

var (
	serveAddr          string
	serveTokenFile     string
	servePublic        bool
	serveAllowSources  []string
	serveMaxConcurrent int
	serveWorkdir       string
)

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().StringVar(&serveAddr, "addr", "127.0.0.1:8080", "address to listen on (use :8080 to listen on all interfaces)")
	serveCmd.Flags().StringVar(&serveTokenFile, "token-file", "", "file of accepted bearer tokens, one per line")
	serveCmd.Flags().BoolVar(&servePublic, "public", false, "serve casts without authentication")
	serveCmd.Flags().StringArrayVar(&serveAllowSources, "allow-source", nil, "only cast molds under this host/owner[/repo] prefix (repeatable)")
	serveCmd.Flags().IntVar(&serveMaxConcurrent, "max-concurrent", 8, "maximum casts rendered at once; extra requests get 503")
	serveCmd.Flags().StringVar(&serveWorkdir, "workdir", "", "working directory for the server process (default: a fresh temp dir)")
}

func runServe(cmd *cobra.Command, _ []string) error {
	tokens, err := loadServeTokens(serveTokenFile, os.Getenv("AILLOY_SERVE_TOKENS"))
	if err != nil {
		return err
	}
	if len(tokens) > 0 && servePublic {
		return errors.New("--public and tokens are mutually exclusive; drop one")
	}

	// Run from an empty directory so cwd-relative lookups (ailloy.lock,
	// .ailloy/) can never leak server state into a response.
	workdir := serveWorkdir
	if workdir == "" {
		if workdir, err = os.MkdirTemp("", "ailloy-serve-"); err != nil {
			return err
		}
		defer func() { _ = os.RemoveAll(workdir) }()
	}
	if err := os.Chdir(workdir); err != nil {
		return fmt.Errorf("entering workdir: %w", err)
	}

	logger := log.New(cmd.ErrOrStderr(), "ailloy serve: ", log.LstdFlags)
	handler, err := server.New(server.Config{
		Renderer:      &serveRenderer{allowSources: serveAllowSources, logger: logger},
		Tokens:        tokens,
		Public:        servePublic,
		MaxConcurrent: serveMaxConcurrent,
		Version:       evolveCurrentVersion,
		Logger:        logger,
	})
	if err != nil {
		return fmt.Errorf("%w (pass --token-file, set AILLOY_SERVE_TOKENS, or pass --public)", err)
	}

	srv := &http.Server{
		Addr:              serveAddr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      5 * time.Minute,
		IdleTimeout:       2 * time.Minute,
	}

	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()

	mode := "token auth"
	if servePublic {
		mode = "PUBLIC, no auth"
	}
	logger.Printf("listening on %s (%s)", serveAddr, mode)

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

// loadServeTokens reads bearer tokens from a file (one per line, blank lines
// and # comments ignored) and a comma-separated environment value.
func loadServeTokens(file, env string) ([]string, error) {
	var tokens []string
	if file != "" {
		f, err := os.Open(file) // #nosec G304 -- operator-supplied token file
		if err != nil {
			return nil, fmt.Errorf("reading token file: %w", err)
		}
		defer func() { _ = f.Close() }()
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			tokens = append(tokens, line)
		}
		if err := sc.Err(); err != nil {
			return nil, fmt.Errorf("reading token file: %w", err)
		}
	}
	for _, t := range strings.Split(env, ",") {
		if t = strings.TrimSpace(t); t != "" {
			tokens = append(tokens, t)
		}
	}
	return tokens, nil
}
