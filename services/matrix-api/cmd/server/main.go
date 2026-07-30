package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v3"
	"github.com/interseguro/matrix-api/internal/adapters/analytics"
	"github.com/interseguro/matrix-api/internal/adapters/auth"
	"github.com/interseguro/matrix-api/internal/adapters/httpapi"
	"github.com/interseguro/matrix-api/internal/application"
	"github.com/interseguro/matrix-api/internal/config"
)

func main() {
	settings, err := config.Load()
	if err != nil {
		log.Fatalf("no se pudo cargar la configuración: %v", err)
	}
	authService, err := auth.New(settings.JWT, settings.Demo)
	if err != nil {
		log.Fatalf("no se pudo inicializar la autenticación: %v", err)
	}
	analyticsClient, err := analytics.NewHTTPClient(settings.Analytics.BaseURL, settings.Analytics.Timeout)
	if err != nil {
		log.Fatalf("no se pudo inicializar el cliente de estadísticas: %v", err)
	}
	processor := application.NewProcessor(settings.Limits, analyticsClient)
	app := httpapi.New(settings.Limits, authService, processor, httpapi.Config{
		BodyLimit:        settings.BodyLimit,
		ReadTimeout:      settings.ReadTimeout,
		WriteTimeout:     settings.WriteTimeout,
		IdleTimeout:      settings.IdleTimeout,
		AuthRateLimitMax: settings.AuthRateLimitMax,
		AuthRateWindow:   settings.AuthRateWindow,
	})

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- app.Listen(settings.Address, fiber.ListenConfig{
			DisableStartupMessage: true,
		})
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-serverErrors:
		if err != nil {
			log.Fatalf("falló el servidor HTTP: %v", err)
		}
	case <-signals:
		ctx, cancel := context.WithTimeout(context.Background(), settings.ShutdownTimeout)
		defer cancel()
		if err := app.ShutdownWithContext(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("falló el apagado controlado: %v", err)
		}
	}
}
