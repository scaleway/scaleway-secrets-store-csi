package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/scaleway/scaleway-secrets-store-csi/internal/provider"
	"github.com/scaleway/scaleway-secrets-store-csi/internal/server"
	"github.com/scaleway/scaleway-secrets-store-csi/internal/version"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	pb "sigs.k8s.io/secrets-store-csi-driver/provider/v1alpha1"
)

const (
	defaultEndpoint = "/etc/kubernetes/secrets-store-csi-providers/scaleway.sock"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("error running provider: %v", err.Error())
	}
}

func run() error {
	endpoint := flag.String("endpoint", defaultEndpoint, "path to socket on which to listen for driver gRPC calls")
	healthAddr := flag.String("health-address", ":8080", "configure http listener for reporting health")
	printVersion := flag.Bool("version", false, "prints the version information")
	debug := flag.Bool("debug", false, "sets log to debug level")
	flag.Parse()

	// Create logger
	logLevel := slog.LevelInfo
	if *debug {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))

	// Print version and exit
	if *printVersion {
		v, err := version.Get().MarshalJSON()
		if err != nil {
			return fmt.Errorf("failed to print version: %w", err)
		}

		_, err = fmt.Println(string(v))
		return err
	}

	// Create and listen on unix socket
	listener, err := listen(*endpoint)
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}
	defer func() { _ = listener.Close() }()

	// Create gRPC server
	loggingOpts := []logging.Option{
		logging.WithLogOnEvents(logging.StartCall, logging.FinishCall),
	}
	grpcServer := grpc.NewServer(grpc.ChainUnaryInterceptor(
		logging.UnaryServerInterceptor(interceptorLogger(logger), loggingOpts...),
		recovery.UnaryServerInterceptor(recovery.WithRecoveryHandler(recoveryHandler)),
	))

	// Create HTTP server
	http.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	httpServer := &http.Server{Addr: *healthAddr}

	// Instantiate provider and server
	prov := provider.NewProvider(provider.WithLogger(logger))
	srv := server.NewServer(prov, server.WithLogger(logger))
	pb.RegisterCSIDriverProviderServer(grpcServer, srv)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// Start gRPC server
	go func() {
		logger.Info("starting grpc server", slog.String("endpoint", *endpoint))
		_ = grpcServer.Serve(listener) // always return a non-nil error
		logger.Info("grpc server stopped")
	}()

	// Start HTTP server
	go func() {
		logger.Info("starting http server", slog.String("address", *healthAddr))
		_ = httpServer.ListenAndServe() // always return a non-nil error
		logger.Info("http server stopped")
	}()

	// Wait for an interrupt signal
	<-ctx.Done()

	// Stop HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Warn("failed to shutdown http server", slog.String("err", err.Error()))
	}

	// Stop gRPC server
	grpcServer.GracefulStop()

	return nil
}

func listen(endpoint string) (net.Listener, error) {
	// Check if a unix socket already exists
	_, err := os.Stat(endpoint)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to check for existence of unix socket: %w", err)
	}

	// If there is a unix socket, we remove it
	if err == nil {
		if err := os.Remove(endpoint); err != nil {
			return nil, fmt.Errorf("failed to remove existing unix socket: %w", err)
		}
	}

	// Create the unix socket
	listener, err := net.Listen("unix", endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on unix socket at %s: %w", endpoint, err)
	}

	return listener, nil
}

func interceptorLogger(l *slog.Logger) logging.Logger {
	return logging.LoggerFunc(func(ctx context.Context, lvl logging.Level, msg string, fields ...any) {
		l.Log(ctx, slog.Level(lvl), msg, fields...)
	})
}

func recoveryHandler(p any) error {
	return status.Error(codes.Internal, "internal error")
}
