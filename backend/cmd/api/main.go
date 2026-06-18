// Command api is the entrypoint that wires the layers into the single runnable
// binary. It owns nothing but composition and process lifecycle: load config,
// open the store, construct the dependencies, and serve the httpapi handler
// until a termination signal arrives, then shut down gracefully.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"coffeemug/backend/internal/auth"
	"coffeemug/backend/internal/config"
	"coffeemug/backend/internal/httpapi"
	"coffeemug/backend/internal/paystack"
	"coffeemug/backend/internal/sse"
	"coffeemug/backend/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := run(logger); err != nil {
		logger.Error("fatal", "error", err)
		os.Exit(1)
	}
}

// run holds the real startup logic so every failure path returns an error to a
// single place (main) rather than calling os.Exit from deep in the wiring,
// which would skip deferred cleanup.
func run(logger *slog.Logger) error {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		return err
	}

	st, err := store.Open(cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer st.Close()

	// Fail fast: a server that boots with an unreachable database is worse than
	// one that refuses to start, because it looks healthy until the first request.
	pingCtx, cancelPing := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelPing()
	if err := st.DB().PingContext(pingCtx); err != nil {
		return err
	}

	tokens := auth.NewTokenManager(cfg.JWTSecret)
	payments := paystack.NewClient(cfg.PaystackSecretKey, cfg.PaystackBaseURL)
	broker := sse.NewBroker()

	srv := httpapi.NewServer(cfg, st, tokens, payments, broker, logger)

	httpSrv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: srv.Handler(),
		// ReadHeaderTimeout/ReadTimeout bound how long a slow client can hold a
		// request open (slowloris protection). WriteTimeout is deliberately 0:
		// the SSE order-tracking endpoint streams for the lifetime of the order,
		// and any positive write deadline would sever those long-lived responses.
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      0,
		IdleTimeout:       120 * time.Second,
	}

	// Serve in the background so main can wait on a termination signal.
	serveErr := make(chan error, 1)
	go func() {
		logger.Info("listening", "port", cfg.Port)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
			return
		}
		serveErr <- nil
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serveErr:
		// The listener fell over on its own (e.g. port already in use).
		return err
	case <-ctx.Done():
		stop() // restore default signal handling so a second Ctrl-C kills us hard
		logger.Info("shutting down")
		// Give in-flight requests a bounded window to finish. Shutdown closes the
		// listener and waits; the SSE loops exit when their request contexts are
		// cancelled at the deadline.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return httpSrv.Shutdown(shutdownCtx)
	}
}
