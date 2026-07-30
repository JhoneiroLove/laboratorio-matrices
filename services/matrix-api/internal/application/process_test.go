package application

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/interseguro/matrix-api/internal/domain"
)

type fakeAnalytics struct {
	token     string
	requestID string
	matrices  []domain.NamedMatrix
	result    domain.StatisticsResult
	err       error
	readyErr  error
}

func (fake *fakeAnalytics) Analyze(_ context.Context, token, requestID string, matrices []domain.NamedMatrix) (domain.StatisticsResult, error) {
	fake.token, fake.requestID, fake.matrices = token, requestID, matrices
	return fake.result, fake.err
}

func (fake *fakeAnalytics) Ready(context.Context) error {
	return fake.readyErr
}

func TestProcessorReady(t *testing.T) {
	wantErr := errors.New("servicio de estadísticas no disponible")
	tests := []struct {
		name string
		err  error
	}{
		{name: "disponible"},
		{name: "dependencia caída", err: wantErr},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			processor := NewProcessor(domain.Limits{}, &fakeAnalytics{readyErr: test.err})
			err := processor.Ready(context.Background())
			if !errors.Is(err, test.err) {
				t.Fatalf("Ready() devolvió el error %v; se esperaba %v", err, test.err)
			}
		})
	}
}

func TestProcessorProcess(t *testing.T) {
	statistics := domain.StatisticsResult{
		Global:   domain.StatisticsSummary{Minimum: 1, Maximum: 6, Sum: 21, Average: 3.5, Elements: 6},
		Matrices: []domain.MatrixStatistics{{Name: "rotated", Minimum: 1, Maximum: 6, Sum: 21, Average: 3.5, Elements: 6}},
	}
	fake := &fakeAnalytics{result: statistics}
	processor := NewProcessor(domain.Limits{MaxRows: 10, MaxColumns: 10, MaxElements: 100}, fake)
	matrix := domain.Matrix{{1, 2}, {3, 4}, {5, 6}}

	result, err := processor.Process(context.Background(), matrix, "Bearer token", "request-123")
	if err != nil {
		t.Fatalf("Process() devolvió un error: %v", err)
	}
	if fake.token != "Bearer token" {
		t.Fatalf("token reenviado = %q", fake.token)
	}
	if fake.requestID != "request-123" {
		t.Fatalf("ID de solicitud reenviado = %q", fake.requestID)
	}
	wantMatrices := []domain.NamedMatrix{
		{Name: "rotated", Values: result.Rotation.Matrix},
		{Name: "Q", Values: result.QR.Q},
		{Name: "R", Values: result.QR.R},
	}
	if !reflect.DeepEqual(fake.matrices, wantMatrices) {
		t.Fatalf("matrices de estadísticas = %#v; se esperaban los resultados de proceso con nombre", fake.matrices)
	}
	if result.Rotation.Direction != "clockwise" || !reflect.DeepEqual(result.Statistics, statistics) {
		t.Fatalf("resultado de Process() = %#v", result)
	}
}

func TestProcessorDoesNotCallAnalyticsForInvalidMatrix(t *testing.T) {
	fake := &fakeAnalytics{}
	processor := NewProcessor(domain.Limits{MaxRows: 1, MaxColumns: 1, MaxElements: 1}, fake)
	_, err := processor.Process(context.Background(), domain.Matrix{{1, 2}}, "Bearer token", "request-123")
	if !errors.Is(err, domain.ErrInvalidMatrix) {
		t.Fatalf("Process() devolvió el error %v; se esperaba ErrInvalidMatrix", err)
	}
	if fake.matrices != nil {
		t.Fatal("se llamó al servicio de estadísticas con una entrada no válida")
	}
}

func TestProcessorReturnsAnalyticsError(t *testing.T) {
	wantErr := errors.New("servicio de estadísticas no disponible")
	processor := NewProcessor(domain.Limits{MaxRows: 2, MaxColumns: 2, MaxElements: 4}, &fakeAnalytics{err: wantErr})
	_, err := processor.Process(context.Background(), domain.Matrix{{1}}, "Bearer token", "request-123")
	if !errors.Is(err, wantErr) {
		t.Fatalf("Process() devolvió el error %v; se esperaba %v", err, wantErr)
	}
}
