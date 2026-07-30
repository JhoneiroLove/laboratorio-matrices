package application

import (
	"context"
	"fmt"

	"github.com/interseguro/matrix-api/internal/domain"
	"github.com/interseguro/matrix-api/internal/ports"
)

type ProcessResult struct {
	Rotation   Rotation                `json:"rotation"`
	QR         QRResult                `json:"qr"`
	Statistics domain.StatisticsResult `json:"statistics"`
}

type Rotation struct {
	Direction string        `json:"direction"`
	Matrix    domain.Matrix `json:"matrix"`
}

type QRResult struct {
	Q domain.Matrix `json:"q"`
	R domain.Matrix `json:"r"`
}

type Processor struct {
	limits    domain.Limits
	analytics ports.AnalyticsClient
}

func NewProcessor(limits domain.Limits, analytics ports.AnalyticsClient) *Processor {
	return &Processor{limits: limits, analytics: analytics}
}

func (processor *Processor) Ready(ctx context.Context) error {
	return processor.analytics.Ready(ctx)
}

func (processor *Processor) Process(ctx context.Context, matrix domain.Matrix, bearerToken, requestID string) (ProcessResult, error) {
	if _, err := domain.Validate(matrix, processor.limits); err != nil {
		return ProcessResult{}, err
	}
	rotated := domain.RotateClockwise(matrix)
	q, r, err := domain.ReducedQR(matrix)
	if err != nil {
		return ProcessResult{}, err
	}
	statistics, err := processor.analytics.Analyze(ctx, bearerToken, requestID, []domain.NamedMatrix{
		{Name: "rotated", Values: rotated},
		{Name: "Q", Values: q},
		{Name: "R", Values: r},
	})
	if err != nil {
		return ProcessResult{}, fmt.Errorf("%w: %w", ports.ErrAnalytics, err)
	}
	return ProcessResult{
		Rotation:   Rotation{Direction: "clockwise", Matrix: rotated},
		QR:         QRResult{Q: q, R: r},
		Statistics: statistics,
	}, nil
}
