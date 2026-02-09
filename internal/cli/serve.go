package cli

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/itsmrshow/bulwark/internal/api"
	"github.com/itsmrshow/bulwark/internal/logging"
)

// NewServeCommand creates the serve command
func NewServeCommand() *cobra.Command {
	cfg := api.LoadConfig()

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run Bulwark Web Console (API + UI)",
		Long: `Runs Bulwark as a daemon process with:
- Web Console (API + UI)
- Scheduler and webhook support (planned)`,
		RunE: runServe,
	}

	cmd.Flags().String("addr", cfg.Addr, "Web UI/API listen address")
	cmd.Flags().String("root", cfg.Root, "Root directory to scan for compose projects")
	cmd.Flags().String("state", cfg.StateDB, "Path to state database (SQLite)")
	cmd.Flags().String("ui-dist", cfg.DistDir, "Path to built UI assets")
	cmd.Flags().Bool("ui-enabled", cfg.UIEnabled, "Enable the web UI")
	cmd.Flags().Bool("ui-readonly", cfg.ReadOnly, "Run UI in read-only mode")

	return cmd
}

func runServe(cmd *cobra.Command, args []string) error {
	logger := logging.Default()
	cfg := api.LoadConfig()

	addr, _ := cmd.Flags().GetString("addr")
	root, _ := cmd.Flags().GetString("root")
	stateFile, _ := cmd.Flags().GetString("state")
	distDir, _ := cmd.Flags().GetString("ui-dist")
	uiEnabled, _ := cmd.Flags().GetBool("ui-enabled")
	uiReadonly, _ := cmd.Flags().GetBool("ui-readonly")

	cfg.Addr = addr
	cfg.Root = root
	cfg.StateDB = stateFile
	cfg.DistDir = distDir
	cfg.UIEnabled = uiEnabled
	cfg.ReadOnly = uiReadonly

	server, err := api.NewServer(cfg, logger)
	if err != nil {
		return err
	}
	defer func() { _ = server.Close() }()

	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Info().Str("addr", cfg.Addr).Msg("Bulwark Web Console listening")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error().Err(err).Msg("server error")
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return httpServer.Shutdown(ctx)
}
