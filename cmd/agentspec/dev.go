package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/szaher/designs/agentz/internal/runtime"
)

func newDevCmd() *cobra.Command {
	var port int

	cmd := &cobra.Command{
		Use:   "dev [file.ias]",
		Short: "Start development server with hot reload",
		Long:  "Watches .ias files for changes and automatically restarts the runtime.",
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

			return runDevLoop(ctx, files, port, logger)
		},
	}

	cmd.Flags().IntVar(&port, "port", 8080, "HTTP server port")

	return cmd
}

func runDevLoop(ctx context.Context, files []string, port int, logger *slog.Logger) error {
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

		rt, err = runtime.New(config, runtime.Options{
			Port:   port,
			Logger: logger,
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
	logger.Info("starting dev server", "files", files, "port", port)
	if err := startRuntime(); err != nil {
		logger.Error("initial start failed", "error", err)
		return err
	}

	// Watch for file changes
	watchDir := "."
	if len(files) > 0 {
		watchDir = filepath.Dir(files[0])
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	lastMod := time.Now()

	for {
		select {
		case <-ctx.Done():
			logger.Info("shutting down dev server")
			if rt != nil {
				shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer shutdownCancel()
				_ = rt.Shutdown(shutdownCtx)
			}
			return nil

		case <-ticker.C:
			// Check for file modifications
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
				logger.Info("file change detected, restarting...")
				if err := startRuntime(); err != nil {
					logger.Error("restart failed", "error", err)
				} else {
					logger.Info("runtime restarted successfully")
				}
			}
		}
	}
}
