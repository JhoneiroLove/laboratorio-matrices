package analytics

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/interseguro/matrix-api/internal/domain"
	"github.com/interseguro/matrix-api/internal/ports"
)

func TestHTTPClientAnalyze(t *testing.T) {
	want := domain.StatisticsResult{
		Global: domain.StatisticsSummary{Minimum: -1, Maximum: 4, Sum: 9, Average: 1.5, Elements: 6},
		Matrices: []domain.MatrixStatistics{
			{Name: "rotated", Minimum: 1, Maximum: 4, Sum: 10, Average: 2.5, Elements: 4, Diagonal: false},
			{Name: "Q", Minimum: -1, Maximum: 1, Sum: -1, Average: -0.5, Elements: 2, Diagonal: false},
		},
		AnyDiagonal: false,
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/v1/statistics" {
			t.Errorf("ruta = %q", request.URL.Path)
		}
		if got := request.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q", got)
		}
		if got := request.Header.Get("X-Request-ID"); got != "request-123" {
			t.Errorf("X-Request-ID = %q", got)
		}
		var body struct {
			Matrices []domain.NamedMatrix `json:"matrices"`
		}
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Errorf("no se pudo decodificar el cuerpo: %v", err)
		}
		for index, name := range []string{"rotated", "Q", "R"} {
			if len(body.Matrices) != 3 || body.Matrices[index].Name != name {
				t.Errorf("matrices recibidas = %#v", body.Matrices)
				break
			}
		}
		writer.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(writer).Encode(want); err != nil {
			t.Errorf("no se pudo codificar la respuesta: %v", err)
		}
	}))
	defer server.Close()

	client, err := NewHTTPClient(server.URL, time.Second)
	if err != nil {
		t.Fatalf("NewHTTPClient() devolvió un error: %v", err)
	}
	got, err := client.Analyze(context.Background(), "Bearer test-token", "request-123", []domain.NamedMatrix{
		{Name: "rotated", Values: domain.Matrix{{1}}},
		{Name: "Q", Values: domain.Matrix{{1}}},
		{Name: "R", Values: domain.Matrix{{1}}},
	})
	if err != nil {
		t.Fatalf("Analyze() devolvió un error: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Analyze() = %#v; se esperaba %#v", got, want)
	}
}

func TestHTTPClientReady(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		contentType string
		body        string
		wantErr     bool
	}{
		{name: "JSON ok", status: http.StatusOK, contentType: "application/json", body: `{"status":"ok"}`},
		{name: "cuerpo vacío opcional", status: http.StatusOK},
		{name: "dependencia caída", status: http.StatusServiceUnavailable, wantErr: true},
		{name: "estado JSON incorrecto", status: http.StatusOK, contentType: "application/json", body: `{"status":"error"}`, wantErr: true},
		{name: "campo JSON desconocido", status: http.StatusOK, contentType: "application/json", body: `{"status":"ok","extra":true}`, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if request.Method != http.MethodGet || request.URL.Path != "/health/ready" {
					t.Errorf("solicitud = %s %s", request.Method, request.URL.Path)
				}
				if test.contentType != "" {
					writer.Header().Set("Content-Type", test.contentType)
				}
				writer.WriteHeader(test.status)
				_, _ = writer.Write([]byte(test.body))
			}))
			defer server.Close()

			client, err := NewHTTPClient(server.URL, time.Second)
			if err != nil {
				t.Fatalf("NewHTTPClient() devolvió un error: %v", err)
			}
			err = client.Ready(context.Background())
			if (err != nil) != test.wantErr {
				t.Fatalf("Ready() devolvió el error %v; wantErr=%v", err, test.wantErr)
			}
		})
	}
}

func TestHTTPClientAnalyzeRejectsUpstreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		http.Error(writer, "detalles internos", http.StatusInternalServerError)
	}))
	defer server.Close()
	client, err := NewHTTPClient(server.URL, time.Second)
	if err != nil {
		t.Fatalf("NewHTTPClient() devolvió un error: %v", err)
	}
	if _, err := client.Analyze(context.Background(), "Bearer token", "request-123", []domain.NamedMatrix{{Name: "R", Values: domain.Matrix{{1}}}}); err == nil {
		t.Fatal("Analyze() devolvió nil; se esperaba un error de estado del servicio remoto")
	}
}

func TestHTTPClientAnalyzeRejectsMalformedSuccess(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
	}{
		{name: "falta anyDiagonal", contentType: "application/json", body: `{"global":{"minimum":1,"maximum":1,"sum":1,"average":1,"elements":1},"matrices":[{"name":"R","minimum":1,"maximum":1,"sum":1,"average":1,"elements":1,"diagonal":true}]}`},
		{name: "falta un campo de la matriz", contentType: "application/json", body: `{"global":{"minimum":1,"maximum":1,"sum":1,"average":1,"elements":1},"matrices":[{"name":"R","minimum":1,"maximum":1,"sum":1,"average":1,"elements":1}],"anyDiagonal":false}`},
		{name: "campo desconocido", contentType: "application/json", body: `{"global":{"minimum":1,"maximum":1,"sum":1,"average":1,"elements":1},"matrices":[{"name":"R","minimum":1,"maximum":1,"sum":1,"average":1,"elements":1,"diagonal":true}],"anyDiagonal":true,"extra":1}`},
		{name: "Content-Type incorrecto", contentType: "text/plain", body: `{}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.Header().Set("Content-Type", test.contentType)
				_, _ = writer.Write([]byte(test.body))
			}))
			defer server.Close()
			client, err := NewHTTPClient(server.URL, time.Second)
			if err != nil {
				t.Fatalf("NewHTTPClient() devolvió un error: %v", err)
			}
			if _, err := client.Analyze(context.Background(), "Bearer token", "request-123", []domain.NamedMatrix{{Name: "R", Values: domain.Matrix{{1}}}}); err == nil {
				t.Fatal("Analyze() devolvió nil; se esperaba el rechazo de una respuesta mal formada")
			}
		})
	}
}

func TestHTTPClientAnalyzeReportsTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		time.Sleep(50 * time.Millisecond)
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{}`))
	}))
	defer server.Close()
	client, err := NewHTTPClient(server.URL, time.Millisecond)
	if err != nil {
		t.Fatalf("NewHTTPClient() devolvió un error: %v", err)
	}
	_, err = client.Analyze(context.Background(), "Bearer token", "request-123", []domain.NamedMatrix{{Name: "R", Values: domain.Matrix{{1}}}})
	if !errors.Is(err, ports.ErrAnalyticsTimeout) {
		t.Fatalf("Analyze() devolvió el error %v; se esperaba ErrAnalyticsTimeout", err)
	}
}
