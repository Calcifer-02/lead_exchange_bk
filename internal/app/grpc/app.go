package grpcapp

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"lead_exchange/internal/grpc/authgrpc"
	"lead_exchange/internal/grpc/dealgrpc"
	"lead_exchange/internal/grpc/filegrpc"
	"lead_exchange/internal/grpc/leadgrpc"
	"lead_exchange/internal/grpc/propertygrpc"
	"lead_exchange/internal/grpc/usergrpc"
	minio "lead_exchange/internal/lib/minio/core"
	"lead_exchange/internal/middleware"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	pb "lead_exchange/pkg"

	"github.com/go-chi/chi/v5"
	"github.com/rs/cors"
	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type App struct {
	log        *slog.Logger
	gRPCServer *grpc.Server
	port       int
}

// New создаёт gRPC + HTTP (Gateway) сервер с Auth, User, File, Lead, Deal и Property сервисами.
func New(
	log *slog.Logger,
	authSvc authgrpc.AuthService,
	userSvc usergrpc.UserService,
	minioClient minio.Client,
	leadSvc leadgrpc.LeadService,
	dealSvc dealgrpc.DealService,
	propertySvc propertygrpc.PropertyService,
	port int,
	secret string,
	disableAuth bool,
) *App {
	loggingOpts := []logging.Option{
		logging.WithLogOnEvents(logging.PayloadReceived, logging.PayloadSent),
	}
	recoveryOpts := []recovery.Option{
		recovery.WithRecoveryHandler(func(p interface{}) (err error) {
			log.Error("Recovered from panic", slog.Any("panic", p))
			return status.Errorf(codes.Internal, "internal error")
		}),
	}

	interceptors := []grpc.UnaryServerInterceptor{
		recovery.UnaryServerInterceptor(recoveryOpts...),
		logging.UnaryServerInterceptor(InterceptorLogger(log), loggingOpts...),
	}

	// Добавляем JWT interceptor только если auth не отключен
	if !disableAuth {
		interceptors = append(interceptors, middleware.JWTUnaryInterceptor(secret, false))
	} else {
		log.Warn("Authentication is DISABLED - all requests will use test user ID")
		// Когда auth отключен, используем interceptor который всегда пропускает с тестовым user ID
		interceptors = append(interceptors, middleware.JWTUnaryInterceptor(secret, true))
	}

	gRPCServer := grpc.NewServer(grpc.ChainUnaryInterceptor(interceptors...))

	// ✅ Регистрируем все gRPC сервера
	authgrpc.RegisterAuthServerGRPC(gRPCServer, authSvc)
	usergrpc.RegisterUserServerGRPC(gRPCServer, userSvc)
	leadgrpc.RegisterLeadServerGRPC(gRPCServer, leadSvc)
	dealgrpc.RegisterDealServerGRPC(gRPCServer, dealSvc, userSvc)
	propertygrpc.RegisterPropertyServerGRPC(gRPCServer, propertySvc)
	if minioClient != nil {
		filegrpc.RegisterFileServerGRPC(gRPCServer, minioClient)
	}

	return &App{
		log:        log,
		gRPCServer: gRPCServer,
		port:       port,
	}
}

func InterceptorLogger(l *slog.Logger) logging.Logger {
	return logging.LoggerFunc(func(ctx context.Context, lvl logging.Level, msg string, fields ...any) {
		l.Log(ctx, slog.Level(lvl), msg, fields...)
	})
}

// mergeSwaggerFiles объединяет несколько swagger JSON файлов в один документ.
func mergeSwaggerFiles(filePaths []string) ([]byte, error) {
	type SwaggerDoc struct {
		Swagger     string                 `json:"swagger"`
		Info        map[string]interface{} `json:"info"`
		Tags        []interface{}          `json:"tags"`
		Consumes    []string               `json:"consumes"`
		Produces    []string               `json:"produces"`
		Paths       map[string]interface{} `json:"paths"`
		Definitions map[string]interface{} `json:"definitions"`
	}

	merged := SwaggerDoc{
		Swagger:     "2.0",
		Info:        map[string]interface{}{"title": "Lead Exchange API", "version": "1.0"},
		Tags:        []interface{}{},
		Consumes:    []string{"application/json"},
		Produces:    []string{"application/json"},
		Paths:       make(map[string]interface{}),
		Definitions: make(map[string]interface{}),
	}

	seenTags := make(map[string]bool)

	for _, filePath := range filePaths {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", filePath, err)
		}

		var doc SwaggerDoc
		if err := json.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", filePath, err)
		}

		// Объединяем теги
		for _, tag := range doc.Tags {
			if tagMap, ok := tag.(map[string]interface{}); ok {
				if name, ok := tagMap["name"].(string); ok && !seenTags[name] {
					merged.Tags = append(merged.Tags, tag)
					seenTags[name] = true
				}
			}
		}

		// Объединяем пути
		for path, methods := range doc.Paths {
			merged.Paths[path] = methods
		}

		// Объединяем определения
		for defName, def := range doc.Definitions {
			merged.Definitions[defName] = def
		}
	}

	result, err := json.Marshal(merged)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal merged swagger: %w", err)
	}

	return result, nil
}

func (a *App) MustRun() {
	if err := a.Run(); err != nil {
		panic(err)
	}
}

func (a *App) Run() error {
	const op = "grpcapp.Run"

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 2)

	// === gRPC Server ===
	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", a.port))
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	// === HTTP Gateway ===
	gwMux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	for _, register := range []func(context.Context, *runtime.ServeMux, string, []grpc.DialOption) error{
		pb.RegisterAuthServiceHandlerFromEndpoint,
		pb.RegisterUserServiceHandlerFromEndpoint,
		pb.RegisterFileServiceHandlerFromEndpoint,
		pb.RegisterLeadServiceHandlerFromEndpoint,
		pb.RegisterDealServiceHandlerFromEndpoint,
		pb.RegisterPropertyServiceHandlerFromEndpoint,
	} {
		if err := register(ctx, gwMux, fmt.Sprintf("localhost:%d", a.port), opts); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	// === Swagger UI ===
	httpMux := http.NewServeMux()
	httpMux.Handle("/", gwMux)

	swaggerMux := chi.NewMux()

	swaggerFiles := []string{
		"pkg/auth.swagger.json",
		"pkg/user.swagger.json",
		"pkg/file.swagger.json",
		"pkg/lead.swagger.json",
		"pkg/deal.swagger.json",
		"pkg/property.swagger.json",
	}

	// Объединённый swagger.json со всеми сервисами
	swaggerMux.HandleFunc("/swagger/doc.json", func(w http.ResponseWriter, _ *http.Request) {
		merged, err := mergeSwaggerFiles(swaggerFiles)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to merge swagger files: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(merged)
	})

	// Отдаём swagger.json для каждого сервиса отдельно (для обратной совместимости)
	swaggerFileMap := map[string]string{
		"/swagger/auth/doc.json":     "pkg/auth.swagger.json",
		"/swagger/user/doc.json":      "pkg/user.swagger.json",
		"/swagger/file/doc.json":      "pkg/file.swagger.json",
		"/swagger/lead/doc.json":      "pkg/lead.swagger.json",
		"/swagger/deal/doc.json":      "pkg/deal.swagger.json",
		"/swagger/property/doc.json":  "pkg/property.swagger.json",
	}

	for route, path := range swaggerFileMap {
		swPath := path // копия для замыкания
		swRoute := route
		swaggerMux.HandleFunc(swRoute, func(w http.ResponseWriter, _ *http.Request) {
			b, err := os.ReadFile(swPath)
			if err != nil {
				http.Error(w, fmt.Sprintf("swagger file not found: %s", swPath), http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(b)
		})
	}

	// Swagger UI с объединённым документом
	swaggerMux.HandleFunc("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	httpMux.Handle("/swagger/", swaggerMux)

	httpServer := &http.Server{
		Addr:         ":8081",
		Handler:      cors.AllowAll().Handler(httpMux),
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}

	// === Запускаем gRPC ===
	go func() {
		a.log.Info("gRPC server started", slog.String("addr", grpcListener.Addr().String()))
		if err := a.gRPCServer.Serve(grpcListener); err != nil && err != grpc.ErrServerStopped {
			errCh <- fmt.Errorf("gRPC server error: %w", err)
		}
	}()

	// === Запускаем HTTP Gateway ===
	go func() {
		a.log.Info("HTTP server started", slog.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
		a.log.Info("shutdown signal received")
	case err := <-errCh:
		cancel()
		return fmt.Errorf("%s: %w", op, err)
	}

	a.log.Info("shutting down servers")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	var shutdownErr error
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		shutdownErr = fmt.Errorf("HTTP shutdown error: %w", err)
	}
	a.gRPCServer.GracefulStop()

	return shutdownErr
}

func (a *App) Stop() {
	a.log.Info("stopping gRPC server", slog.Int("port", a.port))
	a.gRPCServer.GracefulStop()
}
