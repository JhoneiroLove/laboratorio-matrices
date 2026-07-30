package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	recoverer "github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/google/uuid"
	"github.com/interseguro/matrix-api/internal/adapters/auth"
	"github.com/interseguro/matrix-api/internal/application"
	"github.com/interseguro/matrix-api/internal/domain"
	"github.com/interseguro/matrix-api/internal/ports"
)

const requestIDLocal = "request_id"

type API struct {
	limits    domain.Limits
	auth      *auth.Service
	processor *application.Processor
}

type Config struct {
	BodyLimit        int
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	IdleTimeout      time.Duration
	AuthRateLimitMax int
	AuthRateWindow   time.Duration
}

func New(limits domain.Limits, authService *auth.Service, processor *application.Processor, config Config) *fiber.App {
	api := &API{limits: limits, auth: authService, processor: processor}
	app := fiber.New(fiber.Config{
		BodyLimit:    config.BodyLimit,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		IdleTimeout:  config.IdleTimeout,
		ErrorHandler: api.handleError,
	})
	app.Use(api.requestID)
	app.Use(helmet.New())
	app.Use(recoverer.New())

	app.Get("/health/live", api.live)
	app.Get("/health/ready", api.ready)
	app.Post("/api/v1/auth/token", limiter.New(limiter.Config{
		Max:        config.AuthRateLimitMax,
		Expiration: config.AuthRateWindow,
		KeyGenerator: func(c fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c fiber.Ctx) error {
			return tooManyRequests("se superó el límite de solicitudes de token")
		},
	}), api.token)

	protected := app.Group("/api/v1/matrices", api.requireJWT)
	protected.Post("/rotate", api.rotate)
	protected.Post("/qr", api.qr)
	protected.Post("/process", api.process)
	return app
}

type matrixRequest struct {
	Matrix domain.Matrix `json:"matrix"`
}

func (api *API) live(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}

func (api *API) ready(c fiber.Ctx) error {
	if err := api.processor.Ready(c.Context()); err != nil {
		return serviceUnavailable("el servicio de estadísticas no está disponible")
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (api *API) token(c fiber.Ctx) error {
	var request struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSON(c, &request); err != nil {
		return badRequest(err.Error())
	}
	if strings.TrimSpace(request.Username) == "" || strings.TrimSpace(request.Password) == "" {
		return badRequest("username y password no pueden estar vacíos")
	}
	if !api.auth.Authenticate(request.Username, request.Password) {
		c.Set("WWW-Authenticate", "Bearer")
		return unauthorized("las credenciales no son válidas")
	}
	token, expiresIn, err := api.auth.Issue(request.Username)
	if err != nil {
		return err
	}
	c.Set(fiber.HeaderCacheControl, "no-store")
	return c.JSON(fiber.Map{
		"accessToken": token,
		"tokenType":   "Bearer",
		"expiresIn":   expiresIn,
	})
}

func (api *API) rotate(c fiber.Ctx) error {
	request, err := api.matrixRequest(c)
	if err != nil {
		return err
	}
	return c.JSON(application.Rotation{Direction: "clockwise", Matrix: domain.RotateClockwise(request.Matrix)})
}

func (api *API) qr(c fiber.Ctx) error {
	request, err := api.matrixRequest(c)
	if err != nil {
		return err
	}
	q, r, err := domain.ReducedQR(request.Matrix)
	if err != nil {
		return err
	}
	return c.JSON(application.QRResult{Q: q, R: r})
}

func (api *API) process(c fiber.Ctx) error {
	request, err := api.matrixRequest(c)
	if err != nil {
		return err
	}
	result, err := api.processor.Process(c.Context(), request.Matrix, c.Get("Authorization"), requestID(c))
	if err != nil {
		return err
	}
	return c.JSON(struct {
		RequestID  string                  `json:"requestId"`
		Rotation   application.Rotation    `json:"rotation"`
		QR         application.QRResult    `json:"qr"`
		Statistics domain.StatisticsResult `json:"statistics"`
	}{
		RequestID: requestID(c), Rotation: result.Rotation,
		QR: result.QR, Statistics: result.Statistics,
	})
}

func (api *API) matrixRequest(c fiber.Ctx) (matrixRequest, error) {
	var request matrixRequest
	if err := decodeJSON(c, &request); err != nil {
		return request, badRequest(err.Error())
	}
	if _, err := domain.Validate(request.Matrix, api.limits); err != nil {
		return request, err
	}
	return request, nil
}

func (api *API) requireJWT(c fiber.Ctx) error {
	header := c.Get("Authorization")
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		c.Set("WWW-Authenticate", "Bearer")
		return unauthorized("se requiere un bearer token")
	}
	if err := api.auth.Verify(strings.TrimSpace(parts[1])); err != nil {
		c.Set("WWW-Authenticate", "Bearer error=\"invalid_token\"")
		return unauthorized("el bearer token no es válido o ha expirado")
	}
	return c.Next()
}

func decodeJSON(c fiber.Ctx, target any) error {
	if encoding := c.Get("Content-Encoding"); encoding != "" && !strings.EqualFold(encoding, "identity") {
		return errors.New("no se admiten cuerpos de solicitud comprimidos")
	}
	mediaType, _, err := mime.ParseMediaType(c.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return errors.New("Content-Type debe ser application/json")
	}
	decoder := json.NewDecoder(bytes.NewReader(c.Body()))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return errors.New("el cuerpo de la solicitud debe contener un objeto JSON válido")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("el cuerpo de la solicitud debe contener exactamente un objeto JSON")
	}
	return nil
}

type apiError struct {
	status int
	title  string
	detail string
}

func (err *apiError) Error() string { return err.detail }

func badRequest(detail string) error {
	return &apiError{status: fiber.StatusBadRequest, title: "Solicitud incorrecta", detail: detail}
}

func unauthorized(detail string) error {
	return &apiError{status: fiber.StatusUnauthorized, title: "No autorizado", detail: detail}
}

func tooManyRequests(detail string) error {
	return &apiError{status: fiber.StatusTooManyRequests, title: "Demasiadas solicitudes", detail: detail}
}

func serviceUnavailable(detail string) error {
	return &apiError{status: fiber.StatusServiceUnavailable, title: "Servicio no disponible", detail: detail}
}

func (api *API) handleError(c fiber.Ctx, err error) error {
	problem := struct {
		Type      string `json:"type"`
		Title     string `json:"title"`
		Status    int    `json:"status"`
		Detail    string `json:"detail"`
		Instance  string `json:"instance"`
		RequestID string `json:"request_id"`
	}{
		Type:      "about:blank",
		Title:     "Error interno del servidor",
		Status:    fiber.StatusInternalServerError,
		Detail:    "ocurrió un error inesperado",
		Instance:  c.OriginalURL(),
		RequestID: requestID(c),
	}

	var known *apiError
	var fiberError *fiber.Error
	switch {
	case errors.As(err, &known):
		problem.Title, problem.Status, problem.Detail = known.title, known.status, known.detail
	case errors.Is(err, domain.ErrInvalidMatrix):
		problem.Title, problem.Status, problem.Detail = "Contenido no procesable", fiber.StatusUnprocessableEntity, err.Error()
	case errors.Is(err, domain.ErrNumericalRange):
		problem.Title, problem.Status, problem.Detail = "Contenido no procesable", fiber.StatusUnprocessableEntity, err.Error()
	case errors.As(err, &fiberError):
		problem.Status = fiberError.Code
		problem.Title, problem.Detail = localizedHTTPError(fiberError.Code)
	case errors.Is(err, ports.ErrAnalyticsTimeout):
		problem.Title, problem.Status, problem.Detail = "Tiempo de espera del gateway agotado", fiber.StatusGatewayTimeout, "el servicio de estadísticas superó el tiempo de espera"
	case errors.Is(err, ports.ErrAnalytics):
		problem.Title, problem.Status, problem.Detail = "Error de gateway", fiber.StatusBadGateway, "falló la solicitud al servicio de estadísticas"
	}

	return c.Status(problem.Status).JSON(problem, "application/problem+json")
}

func localizedHTTPError(status int) (string, string) {
	switch status {
	case fiber.StatusNotFound:
		return "No encontrado", "no se encontró el recurso solicitado"
	case fiber.StatusMethodNotAllowed:
		return "Método no permitido", "el método HTTP no está permitido para este recurso"
	case fiber.StatusRequestEntityTooLarge:
		return "Contenido demasiado grande", "el cuerpo de la solicitud supera el límite permitido"
	case fiber.StatusUnsupportedMediaType:
		return "Tipo de contenido no admitido", "el tipo de contenido de la solicitud no está admitido"
	default:
		return "Error HTTP", "no se pudo procesar la solicitud HTTP"
	}
}

func (api *API) requestID(c fiber.Ctx) error {
	id := uuid.NewString()
	c.Locals(requestIDLocal, id)
	c.Set("X-Request-ID", id)
	return c.Next()
}

func requestID(c fiber.Ctx) string {
	id, _ := c.Locals(requestIDLocal).(string)
	return id
}
