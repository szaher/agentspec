package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
	"github.com/szaher/agentspec/internal/runtime"
)

func newRunCmd() *cobra.Command {
	var port int
	var enableUI bool
	var noAuth bool
	var corsOrigins string

	cmd := &cobra.Command{
		Use:   "run [file.ias]",
		Short: "Start agent runtime server with hot reload",
		Long:  "Watches .ias files for changes and automatically restarts the runtime. Includes built-in web UI.",
		RunE: func(cmd *cobra.Command, args []string) error {
			files, err := resolveFiles(args)
			if err != nil {
				return err
			}

			logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			}))

			ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer cancel()

			return runServerLoop(ctx, files, port, enableUI, noAuth, corsOrigins, logger)
		},
	}

	cmd.Flags().IntVar(&port, "port", 8080, "HTTP server port")
	cmd.Flags().BoolVar(&enableUI, "ui", true, "Enable built-in web frontend")
	cmd.Flags().BoolVar(&noAuth, "no-auth", false, "Explicitly allow unauthenticated access (WARNING: insecure)")
	cmd.Flags().StringVar(&corsOrigins, "cors-origins", "", "Comma-separated list of allowed CORS origins")

	return cmd
}

func runServerLoop(ctx context.Context, files []string, port int, enableUI bool, noAuth bool, corsOriginsStr string, logger *slog.Logger) error {
	var rt *runtime.Runtime

	startRuntime := func() error {
		if rt != nil {
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			_ = rt.Shutdown(shutdownCtx)
			rt = nil
		}

		doc, err := parseAndLower(files)
		if err != nil {
			return fmt.Errorf("parse error: %w", err)
		}

		config, err := runtime.FromIR(doc)
		if err != nil {
			return fmt.Errorf("config error: %w", err)
		}

		// Build CORS origins — auto-add localhost in dev mode
		var corsOrigins []string
		if corsOriginsStr != "" {
			for _, o := range strings.Split(corsOriginsStr, ",") {
				corsOrigins = append(corsOrigins, strings.TrimSpace(o))
			}
		}
		// Auto-allow localhost origins for built-in UI
		corsOrigins = append(corsOrigins,
			fmt.Sprintf("http://localhost:%d", port),
			fmt.Sprintf("http://127.0.0.1:%d", port),
		)

		rt, err = runtime.New(config, runtime.Options{
			Port:        port,
			Logger:      logger,
			EnableUI:    enableUI,
			NoAuth:      noAuth,
			CORSOrigins: corsOrigins,
		})
		if err != nil {
			return fmt.Errorf("runtime error: %w", err)
		}

		go func() {
			if err := rt.Start(ctx); err != nil {
				logger.Error("runtime stopped", "error", err)
			}
		}()

		return nil
	}

	// Initial start
	logger.Info("starting server", "files", files, "port", port, "ui", enableUI)
	if err := startRuntime(); err != nil {
		logger.Error("initial start failed", "error", err)
		return err
	}

	// Watch for file changes — try fsnotify first, fall back to polling
	watchDir := "."
	if len(files) > 0 {
		watchDir = filepath.Dir(files[0])
	}

	reload := func() {
		logger.Info("file change detected, restarting...")
		if err := startRuntime(); err != nil {
			logger.Error("restart failed", "error", err)
		} else {
			logger.Info("runtime restarted successfully")
		}
	}

	watcher, fsErr := fsnotify.NewWatcher()
	if fsErr != nil {
		logger.Warn("fsnotify unavailable, falling back to polling (2s interval)", "error", fsErr)
		return watchWithPolling(ctx, watchDir, rt, reload, logger)
	}
	defer func() { _ = watcher.Close() }()

	// Add watch directory
	if err := watcher.Add(watchDir); err != nil {
		logger.Warn("fsnotify watch failed, falling back to polling (2s interval)", "error", err)
		return watchWithPolling(ctx, watchDir, rt, reload, logger)
	}

	// Debounce timer — only trigger reload after 100ms of quiet
	debounce := time.NewTimer(0)
	if !debounce.Stop() {
		<-debounce.C
	}
	debouncing := false

	for {
		select {
		case <-ctx.Done():
			logger.Info("shutting down server")
			if rt != nil {
				shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer shutdownCancel()
				_ = rt.Shutdown(shutdownCtx)
			}
			return nil

		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			// Only react to Write and Create events on .ias files
			if filepath.Ext(event.Name) == ".ias" && (event.Op&(fsnotify.Write|fsnotify.Create)) != 0 {
				if debouncing {
					debounce.Reset(100 * time.Millisecond)
				} else {
					debounce.Reset(100 * time.Millisecond)
					debouncing = true
				}
			}

		case <-debounce.C:
			debouncing = false
			reload()

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			logger.Error("file watcher error", "error", err)
		}
	}
}

// watchWithPolling is the fallback file watcher using polling when fsnotify is unavailable.
func watchWithPolling(ctx context.Context, watchDir string, rt *runtime.Runtime, reload func(), logger *slog.Logger) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	lastMod := time.Now()

	for {
		select {
		case <-ctx.Done():
			logger.Info("shutting down server")
			if rt != nil {
				shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer shutdownCancel()
				_ = rt.Shutdown(shutdownCtx)
			}
			return nil

		case <-ticker.C:
			changed := false
			_ = filepath.Walk(watchDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				if filepath.Ext(path) == ".ias" && info.ModTime().After(lastMod) {
					changed = true
					return filepath.SkipAll
				}
				return nil
			})

			if changed {
				lastMod = time.Now()
				reload()
			}
		}
	}
}
