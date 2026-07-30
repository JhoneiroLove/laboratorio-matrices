package ports

import (
	"context"
	"errors"

	"github.com/interseguro/matrix-api/internal/domain"
)

var ErrAnalytics = errors.New("falló la solicitud de estadísticas")
var ErrAnalyticsTimeout = errors.New("la solicitud de estadísticas superó el tiempo de espera")

type AnalyticsClient interface {
	Analyze(ctx context.Context, bearerToken, requestID string, matrices []domain.NamedMatrix) (domain.StatisticsResult, error)
	Ready(ctx context.Context) error
}
