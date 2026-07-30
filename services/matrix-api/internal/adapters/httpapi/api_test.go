package httpapi

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/interseguro/matrix-api/internal/adapters/auth"
	"github.com/interseguro/matrix-api/internal/application"
	"github.com/interseguro/matrix-api/internal/config"
	"github.com/interseguro/matrix-api/internal/domain"
)

type analyticsStub struct {
	readyErr error
}

func (analyticsStub) Analyze(context.Context, string, string, []domain.NamedMatrix) (domain.StatisticsResult, error) {
	return testStatistics(), nil
}

func (stub analyticsStub) Ready(context.Context) error {
	return stub.readyErr
}

func TestReadinessDependsOnAnalyticsAndLivenessIsLocal(t *testing.T) {
	tests := []struct {
		name         string
		analyticsErr error
		readyStatus  int
	}{
		{name: "estadísticas disponible", readyStatus: http.StatusOK},
		{name: "estadísticas caída", analyticsErr: errors.New("dependencia caída"), readyStatus: http.StatusServiceUnavailable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app := testAppWithAnalytics(t, analyticsStub{readyErr: test.analyticsErr})

			response := perform(t, app, http.MethodGet, "/health/live", "", "")
			if response.StatusCode != http.StatusOK {
				t.Fatalf("GET /health/live devolvió el estado %d", response.StatusCode)
			}

			response = perform(t, app, http.MethodGet, "/health/ready", "", "")
			if test.readyStatus == http.StatusOK {
				if response.StatusCode != http.StatusOK {
					t.Fatalf("GET /health/ready devolvió el estado %d", response.StatusCode)
				}
				var health struct {
					Status string `json:"status"`
				}
				if err := json.NewDecoder(response.Body).Decode(&health); err != nil || health.Status != "ok" {
					t.Fatalf("respuesta de /health/ready = %+v; error = %v", health, err)
				}
				return
			}
			assertProblemDetail(t, response, http.StatusServiceUnavailable, "el servicio de estadísticas no está disponible")
		})
	}
}

func TestHealthAndAuthentication(t *testing.T) {
	app := testApp(t)

	response := perform(t, app, http.MethodGet, "/health/live", "", "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("GET /health/live devolvió el estado %d", response.StatusCode)
	}
	assertSecurityHeaders(t, response)
	if response.Header.Get("X-Request-ID") == "" {
		t.Fatal("la respuesta de /health/live no incluye X-Request-ID")
	}
	if _, err := uuid.Parse(response.Header.Get("X-Request-ID")); err != nil {
		t.Fatalf("X-Request-ID no es un UUID: %v", err)
	}
	var health struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(response.Body).Decode(&health); err != nil || health.Status != "ok" {
		t.Fatalf("respuesta de /health/live = %+v; error = %v", health, err)
	}

	response = perform(t, app, http.MethodPost, "/api/v1/matrices/rotate", `{"matrix":[[1]]}`, "")
	assertSecurityHeaders(t, response)
	assertProblem(t, response, http.StatusUnauthorized)

	token := issueToken(t, app)
	response = perform(t, app, http.MethodPost, "/api/v1/matrices/rotate", `{"matrix":[[1,2],[3,4]]}`, token)
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("POST /rotate devolvió el estado %d; cuerpo=%s", response.StatusCode, body)
	}
	assertSecurityHeaders(t, response)
	var result struct {
		Direction string        `json:"direction"`
		Matrix    domain.Matrix `json:"matrix"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("no se pudo decodificar la respuesta de rotate: %v", err)
	}
	if result.Direction != "clockwise" || result.Matrix[0][0] != 3 || result.Matrix[1][1] != 2 {
		t.Fatalf("rotación = %+v", result)
	}
}

func TestQRAndProcessResponseContracts(t *testing.T) {
	app := testApp(t)
	token := issueToken(t, app)

	response := perform(t, app, http.MethodPost, "/api/v1/matrices/qr", `{"matrix":[[1,2],[3,4]]}`, token)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("POST /qr devolvió el estado %d", response.StatusCode)
	}
	assertSecurityHeaders(t, response)
	var qr map[string]json.RawMessage
	if err := json.NewDecoder(response.Body).Decode(&qr); err != nil {
		t.Fatalf("no se pudo decodificar la respuesta QR: %v", err)
	}
	if _, ok := qr["q"]; !ok {
		t.Fatalf("respuesta QR = %v; falta q", qr)
	}
	if _, ok := qr["r"]; !ok {
		t.Fatalf("respuesta QR = %v; falta r", qr)
	}
	if _, ok := qr["Q"]; ok {
		t.Fatalf("la respuesta QR contiene Q en mayúscula: %v", qr)
	}

	response = perform(t, app, http.MethodPost, "/api/v1/matrices/process", `{"matrix":[[1,2],[3,4]]}`, token)
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("POST /process devolvió el estado %d; cuerpo=%s", response.StatusCode, body)
	}
	assertSecurityHeaders(t, response)
	var result struct {
		RequestID  string                  `json:"requestId"`
		Rotation   application.Rotation    `json:"rotation"`
		QR         application.QRResult    `json:"qr"`
		Statistics domain.StatisticsResult `json:"statistics"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("no se pudo decodificar la respuesta de process: %v", err)
	}
	if _, err := uuid.Parse(result.RequestID); err != nil {
		t.Fatalf("requestId de /process = %q; se esperaba un UUID", result.RequestID)
	}
	if result.RequestID != response.Header.Get("X-Request-ID") {
		t.Fatalf("requestId de /process = %q; header = %q", result.RequestID, response.Header.Get("X-Request-ID"))
	}
	if result.Rotation.Direction != "clockwise" || result.QR.Q == nil || result.QR.R == nil || !reflect.DeepEqual(result.Statistics, testStatistics()) {
		t.Fatalf("respuesta de /process = %#v", result)
	}
}

func TestStrictJSONAndMatrixValidationProblems(t *testing.T) {
	app := testApp(t)
	token := issueToken(t, app)
	tests := []struct {
		name   string
		body   string
		status int
	}{
		{name: "campo desconocido", body: `{"matrix":[[1]],"extra":true}`, status: http.StatusBadRequest},
		{name: "dos objetos", body: `{"matrix":[[1]]}{}`, status: http.StatusBadRequest},
		{name: "matriz irregular", body: `{"matrix":[[1,2],[3]]}`, status: http.StatusUnprocessableEntity},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := perform(t, app, http.MethodPost, "/api/v1/matrices/qr", test.body, token)
			assertProblem(t, response, test.status)
		})
	}
}

func TestTokenValidationAndRateLimit(t *testing.T) {
	app := testApp(t)
	for _, body := range []string{
		`{"username":"","password":"secret"}`,
		`{"username":"demo","password":"   "}`,
	} {
		response := perform(t, app, http.MethodPost, "/api/v1/auth/token", body, "")
		assertProblem(t, response, http.StatusBadRequest)
	}

	response := perform(t, app, http.MethodPost, "/api/v1/auth/token", `{"username":"demo","password":"wrong"}`, "")
	if got := response.Header.Get("WWW-Authenticate"); got != "Bearer" {
		t.Fatalf("WWW-Authenticate = %q; se esperaba Bearer", got)
	}
	assertProblem(t, response, http.StatusUnauthorized)

	limited := testAppWithRateLimit(t, 2)
	for attempt := 0; attempt < 2; attempt++ {
		response = perform(t, limited, http.MethodPost, "/api/v1/auth/token", `{"username":"demo","password":"wrong"}`, "")
		assertProblem(t, response, http.StatusUnauthorized)
	}
	response = perform(t, limited, http.MethodPost, "/api/v1/auth/token", `{"username":"demo","password":"wrong"}`, "")
	assertSecurityHeaders(t, response)
	assertProblem(t, response, http.StatusTooManyRequests)
}

func testApp(t *testing.T) *fiber.App {
	return testAppWithRateLimit(t, 100)
}

func testAppWithRateLimit(t *testing.T, rateLimit int) *fiber.App {
	return testAppWithDependencies(t, rateLimit, analyticsStub{})
}

func testAppWithAnalytics(t *testing.T, analytics analyticsStub) *fiber.App {
	return testAppWithDependencies(t, 100, analytics)
}

func testAppWithDependencies(t *testing.T, rateLimit int, analytics analyticsStub) *fiber.App {
	t.Helper()
	directory := t.TempDir()
	privatePath, publicPath := writeKeys(t, directory)
	authService, err := auth.New(config.JWT{
		PrivateKeyPath: privatePath,
		PublicKeyPath:  publicPath,
		Issuer:         "test-issuer",
		Audience:       "test-audience",
		TTL:            time.Minute,
	}, config.DemoCredentials{Username: "demo", Password: "secret"})
	if err != nil {
		t.Fatalf("auth.New() devolvió un error: %v", err)
	}
	limits := domain.Limits{MaxRows: 10, MaxColumns: 10, MaxElements: 100}
	processor := application.NewProcessor(limits, analytics)
	return New(limits, authService, processor, Config{
		BodyLimit: 4096, ReadTimeout: time.Second, WriteTimeout: time.Second, IdleTimeout: time.Second,
		AuthRateLimitMax: rateLimit, AuthRateWindow: time.Minute,
	})
}

func writeKeys(t *testing.T, directory string) (string, string) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("no se pudo generar la clave RSA: %v", err)
	}
	privateDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("no se pudo serializar la clave privada: %v", err)
	}
	publicDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("no se pudo serializar la clave pública: %v", err)
	}
	privatePath, publicPath := filepath.Join(directory, "private.pem"), filepath.Join(directory, "public.pem")
	if err := os.WriteFile(privatePath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateDER}), 0o600); err != nil {
		t.Fatalf("no se pudo escribir la clave privada: %v", err)
	}
	if err := os.WriteFile(publicPath, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDER}), 0o600); err != nil {
		t.Fatalf("no se pudo escribir la clave pública: %v", err)
	}
	return privatePath, publicPath
}

func issueToken(t *testing.T, app *fiber.App) string {
	t.Helper()
	response := perform(t, app, http.MethodPost, "/api/v1/auth/token", `{"username":"demo","password":"secret"}`, "")
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("la emisión del token devolvió el estado %d; cuerpo=%s", response.StatusCode, body)
	}
	assertSecurityHeaders(t, response)
	var body struct {
		AccessToken string `json:"accessToken"`
		TokenType   string `json:"tokenType"`
		ExpiresIn   int64  `json:"expiresIn"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("no se pudo decodificar el token: %v", err)
	}
	if body.TokenType != "Bearer" || body.ExpiresIn != 60 {
		t.Fatalf("respuesta del token = %+v", body)
	}
	return "Bearer " + body.AccessToken
}

func assertSecurityHeaders(t *testing.T, response *http.Response) {
	t.Helper()
	want := map[string]string{
		"X-Content-Type-Options":       "nosniff",
		"X-Frame-Options":              "SAMEORIGIN",
		"Referrer-Policy":              "no-referrer",
		"Cross-Origin-Resource-Policy": "same-origin",
	}
	for name, value := range want {
		if got := response.Header.Get(name); got != value {
			t.Fatalf("%s = %q; se esperaba %q", name, got, value)
		}
	}
	if got := response.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q; no se esperaba un header CORS", got)
	}
}

func testStatistics() domain.StatisticsResult {
	return domain.StatisticsResult{
		Global: domain.StatisticsSummary{Minimum: -1, Maximum: 4, Sum: 8, Average: 1, Elements: 8},
		Matrices: []domain.MatrixStatistics{
			{Name: "rotated", Minimum: 1, Maximum: 4, Sum: 10, Average: 2.5, Elements: 4},
			{Name: "Q", Minimum: -1, Maximum: 1, Sum: 0, Average: 0, Elements: 2},
			{Name: "R", Minimum: 1, Maximum: 4, Sum: 8, Average: 2, Elements: 2, Diagonal: true},
		},
		AnyDiagonal: true,
	}
}

func perform(t *testing.T, app *fiber.App, method, path, body, authorization string) *http.Response {
	t.Helper()
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}
	response, err := app.Test(request, fiber.TestConfig{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("app.Test() devolvió un error: %v", err)
	}
	return response
}

func assertProblem(t *testing.T, response *http.Response, status int) {
	t.Helper()
	if response.StatusCode != status {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("estado = %d; se esperaba %d; cuerpo=%s", response.StatusCode, status, body)
	}
	if got := response.Header.Get("Content-Type"); got != "application/problem+json" {
		t.Fatalf("Content-Type = %q", got)
	}
	var problem struct {
		Status    int    `json:"status"`
		RequestID string `json:"request_id"`
	}
	if err := json.NewDecoder(response.Body).Decode(&problem); err != nil {
		t.Fatalf("no se pudo decodificar el problema: %v", err)
	}
	if problem.Status != status || problem.RequestID == "" {
		t.Fatalf("problema = %+v", problem)
	}
}

func assertProblemDetail(t *testing.T, response *http.Response, status int, detail string) {
	t.Helper()
	if response.StatusCode != status {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("estado = %d; se esperaba %d; cuerpo=%s", response.StatusCode, status, body)
	}
	if got := response.Header.Get("Content-Type"); got != "application/problem+json" {
		t.Fatalf("Content-Type = %q", got)
	}
	var problem struct {
		Status int    `json:"status"`
		Detail string `json:"detail"`
	}
	if err := json.NewDecoder(response.Body).Decode(&problem); err != nil {
		t.Fatalf("no se pudo decodificar el problema: %v", err)
	}
	if problem.Status != status || problem.Detail != detail {
		t.Fatalf("problema = %+v", problem)
	}
}
