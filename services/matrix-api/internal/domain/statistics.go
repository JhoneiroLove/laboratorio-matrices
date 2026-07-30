package domain

type NamedMatrix struct {
	Name   string `json:"name"`
	Values Matrix `json:"values"`
}

type StatisticsSummary struct {
	Minimum  float64 `json:"minimum"`
	Maximum  float64 `json:"maximum"`
	Sum      float64 `json:"sum"`
	Average  float64 `json:"average"`
	Elements int     `json:"elements"`
}

type MatrixStatistics struct {
	Name     string  `json:"name"`
	Minimum  float64 `json:"minimum"`
	Maximum  float64 `json:"maximum"`
	Sum      float64 `json:"sum"`
	Average  float64 `json:"average"`
	Elements int     `json:"elements"`
	Diagonal bool    `json:"diagonal"`
}

type StatisticsResult struct {
	Global      StatisticsSummary  `json:"global"`
	Matrices    []MatrixStatistics `json:"matrices"`
	AnyDiagonal bool               `json:"anyDiagonal"`
}
