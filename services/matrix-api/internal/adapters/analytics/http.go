package analytics

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/interseguro/matrix-api/internal/domain"
	"github.com/interseguro/matrix-api/internal/ports"
)

const maxResponseBytes = 1 << 20

type HTTPClient struct {
	endpoint      string
	readyEndpoint string
	client        *http.Client
}

func NewHTTPClient(baseURL string, timeout time.Duration) (*HTTPClient, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, fmt.Errorf("la URL base del servicio de estadísticas no es válida")
	}
	return &HTTPClient{
		endpoint:      strings.TrimRight(baseURL, "/") + "/api/v1/statistics",
		readyEndpoint: strings.TrimRight(baseURL, "/") + "/health/ready",
		client: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

func (client *HTTPClient) Ready(ctx context.Context) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, client.readyEndpoint, nil)
	if err != nil {
		return fmt.Errorf("no se pudo crear la solicitud de disponibilidad de estadísticas: %w", err)
	}
	response, err := client.client.Do(request)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("%w: %w", ports.ErrAnalyticsTimeout, err)
		}
		return fmt.Errorf("%w: %w", ports.ErrAnalytics, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return fmt.Errorf("%w: el endpoint de disponibilidad devolvió el estado %d", ports.ErrAnalytics, response.StatusCode)
	}
	if err := decodeReady(response); err != nil {
		return fmt.Errorf("%w: %w", ports.ErrAnalytics, err)
	}
	return nil
}

func decodeReady(response *http.Response) error {
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return fmt.Errorf("no se pudo leer la respuesta de disponibilidad: %w", err)
	}
	if len(body) > maxResponseBytes {
		return errors.New("la respuesta de disponibilidad supera el límite de tamaño")
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return nil
	}
	mediaType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return errors.New("la respuesta de disponibilidad debe ser application/json")
	}
	var health struct {
		Status string `json:"status"`
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&health); err != nil || health.Status != "ok" {
		return errors.New("la respuesta de disponibilidad no contiene el estado ok")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("la respuesta de disponibilidad contiene JSON adicional")
	}
	return nil
}

func (client *HTTPClient) Analyze(ctx context.Context, bearerToken, requestID string, matrices []domain.NamedMatrix) (domain.StatisticsResult, error) {
	body, err := json.Marshal(struct {
		Matrices []domain.NamedMatrix `json:"matrices"`
	}{Matrices: matrices})
	if err != nil {
		return domain.StatisticsResult{}, fmt.Errorf("no se pudo codificar la solicitud de estadísticas: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.endpoint, bytes.NewReader(body))
	if err != nil {
		return domain.StatisticsResult{}, fmt.Errorf("no se pudo crear la solicitud de estadísticas: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", bearerToken)
	request.Header.Set("X-Request-ID", requestID)

	response, err := client.client.Do(request)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return domain.StatisticsResult{}, fmt.Errorf("%w: %w", ports.ErrAnalyticsTimeout, err)
		}
		return domain.StatisticsResult{}, fmt.Errorf("%w: %w", ports.ErrAnalytics, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return domain.StatisticsResult{}, fmt.Errorf("%w: el servicio remoto devolvió el estado %d", ports.ErrAnalytics, response.StatusCode)
	}
	mediaType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return domain.StatisticsResult{}, fmt.Errorf("%w: la respuesta remota debe ser application/json", ports.ErrAnalytics)
	}
	statistics, err := decodeStatistics(response.Body)
	if err != nil {
		return domain.StatisticsResult{}, fmt.Errorf("%w: %w", ports.ErrAnalytics, err)
	}
	return statistics, nil
}

type statisticsResponse struct {
	Global      *summaryResponse         `json:"global"`
	Matrices    *[]matrixSummaryResponse `json:"matrices"`
	AnyDiagonal *bool                    `json:"anyDiagonal"`
}

type summaryResponse struct {
	Minimum  *float64 `json:"minimum"`
	Maximum  *float64 `json:"maximum"`
	Sum      *float64 `json:"sum"`
	Average  *float64 `json:"average"`
	Elements *int     `json:"elements"`
}

type matrixSummaryResponse struct {
	Name     *string  `json:"name"`
	Minimum  *float64 `json:"minimum"`
	Maximum  *float64 `json:"maximum"`
	Sum      *float64 `json:"sum"`
	Average  *float64 `json:"average"`
	Elements *int     `json:"elements"`
	Diagonal *bool    `json:"diagonal"`
}

func decodeStatistics(reader io.Reader) (domain.StatisticsResult, error) {
	body, err := io.ReadAll(io.LimitReader(reader, maxResponseBytes+1))
	if err != nil {
		return domain.StatisticsResult{}, fmt.Errorf("no se pudo leer la respuesta remota: %w", err)
	}
	if len(body) > maxResponseBytes {
		return domain.StatisticsResult{}, errors.New("la respuesta remota supera el límite de tamaño")
	}

	var response statisticsResponse
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&response); err != nil {
		return domain.StatisticsResult{}, errors.New("la respuesta remota no contiene JSON de estadísticas válido")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return domain.StatisticsResult{}, errors.New("la respuesta remota contiene JSON adicional")
	}
	if response.Global == nil || response.Matrices == nil || response.AnyDiagonal == nil || len(*response.Matrices) == 0 {
		return domain.StatisticsResult{}, errors.New("a la respuesta remota le faltan campos de estadísticas obligatorios")
	}

	global, err := response.Global.toDomain()
	if err != nil {
		return domain.StatisticsResult{}, err
	}
	matrices := make([]domain.MatrixStatistics, len(*response.Matrices))
	for index, matrix := range *response.Matrices {
		converted, err := matrix.toDomain()
		if err != nil {
			return domain.StatisticsResult{}, fmt.Errorf("el resumen de la matriz %d no es válido: %w", index, err)
		}
		matrices[index] = converted
	}
	return domain.StatisticsResult{Global: global, Matrices: matrices, AnyDiagonal: *response.AnyDiagonal}, nil
}

func (summary summaryResponse) toDomain() (domain.StatisticsSummary, error) {
	if summary.Minimum == nil || summary.Maximum == nil || summary.Sum == nil || summary.Average == nil || summary.Elements == nil {
		return domain.StatisticsSummary{}, errors.New("al resumen de estadísticas le faltan campos obligatorios")
	}
	if !finite(*summary.Minimum) || !finite(*summary.Maximum) || !finite(*summary.Sum) || !finite(*summary.Average) || *summary.Elements < 1 {
		return domain.StatisticsSummary{}, errors.New("el resumen de estadísticas contiene valores no válidos")
	}
	return domain.StatisticsSummary{
		Minimum: *summary.Minimum, Maximum: *summary.Maximum, Sum: *summary.Sum,
		Average: *summary.Average, Elements: *summary.Elements,
	}, nil
}

func (summary matrixSummaryResponse) toDomain() (domain.MatrixStatistics, error) {
	if summary.Name == nil || strings.TrimSpace(*summary.Name) == "" || summary.Minimum == nil || summary.Maximum == nil || summary.Sum == nil || summary.Average == nil || summary.Elements == nil || summary.Diagonal == nil {
		return domain.MatrixStatistics{}, errors.New("al resumen de la matriz le faltan campos obligatorios")
	}
	if !finite(*summary.Minimum) || !finite(*summary.Maximum) || !finite(*summary.Sum) || !finite(*summary.Average) || *summary.Elements < 1 {
		return domain.MatrixStatistics{}, errors.New("el resumen de la matriz contiene valores no válidos")
	}
	return domain.MatrixStatistics{
		Name: *summary.Name, Minimum: *summary.Minimum, Maximum: *summary.Maximum,
		Sum: *summary.Sum, Average: *summary.Average, Elements: *summary.Elements,
		Diagonal: *summary.Diagonal,
	}, nil
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
