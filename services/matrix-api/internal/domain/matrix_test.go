package domain

import (
	"errors"
	"math"
	"reflect"
	"testing"
)

func TestValidate(t *testing.T) {
	limits := Limits{MaxRows: 3, MaxColumns: 3, MaxElements: 6}
	tests := []struct {
		name    string
		matrix  Matrix
		want    Dimensions
		wantErr bool
	}{
		{name: "rectangular válida", matrix: Matrix{{1, 2, 3}, {4, 5, 6}}, want: Dimensions{Rows: 2, Columns: 3}},
		{name: "vacía", matrix: Matrix{}, wantErr: true},
		{name: "fila vacía", matrix: Matrix{{}}, wantErr: true},
		{name: "irregular", matrix: Matrix{{1, 2}, {3}}, wantErr: true},
		{name: "demasiadas filas", matrix: Matrix{{1}, {2}, {3}, {4}}, wantErr: true},
		{name: "demasiadas columnas", matrix: Matrix{{1, 2, 3, 4}}, wantErr: true},
		{name: "demasiados elementos", matrix: Matrix{{1, 2, 3}, {4, 5, 6}, {7, 8, 9}}, wantErr: true},
		{name: "NaN", matrix: Matrix{{math.NaN()}}, wantErr: true},
		{name: "infinito positivo", matrix: Matrix{{math.Inf(1)}}, wantErr: true},
		{name: "infinito negativo", matrix: Matrix{{math.Inf(-1)}}, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := Validate(test.matrix, limits)
			if test.wantErr {
				if !errors.Is(err, ErrInvalidMatrix) {
					t.Fatalf("Validate() devolvió el error %v; se esperaba ErrInvalidMatrix", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Validate() devolvió un error inesperado: %v", err)
			}
			if got != test.want {
				t.Fatalf("Validate() = %+v; se esperaba %+v", got, test.want)
			}
		})
	}
}

func TestRotateClockwise(t *testing.T) {
	tests := []struct {
		name   string
		matrix Matrix
		want   Matrix
	}{
		{name: "cuadrada", matrix: Matrix{{1, 2}, {3, 4}}, want: Matrix{{3, 1}, {4, 2}}},
		{name: "ancha", matrix: Matrix{{1, 2, 3}, {4, 5, 6}}, want: Matrix{{4, 1}, {5, 2}, {6, 3}}},
		{name: "alta", matrix: Matrix{{1, 2}, {3, 4}, {5, 6}}, want: Matrix{{5, 3, 1}, {6, 4, 2}}},
		{name: "único elemento", matrix: Matrix{{7}}, want: Matrix{{7}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			original := clone(test.matrix)
			if got := RotateClockwise(test.matrix); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("RotateClockwise() = %v; se esperaba %v", got, test.want)
			}
			if !reflect.DeepEqual(test.matrix, original) {
				t.Fatal("RotateClockwise() modificó su entrada")
			}
		})
	}
}

func TestReducedQRInvariants(t *testing.T) {
	tests := []struct {
		name   string
		matrix Matrix
	}{
		{name: "cuadrada", matrix: Matrix{{12, -51, 4}, {6, 167, -68}, {-4, 24, -41}}},
		{name: "alta", matrix: Matrix{{1, 2}, {3, 4}, {5, 6}, {7, 8}}},
		{name: "ancha", matrix: Matrix{{1, 2, 3, 4}, {5, 6, 7, 8}}},
		{name: "singular", matrix: Matrix{{1, 2, 3}, {2, 4, 6}, {3, 6, 9}, {4, 8, 12}}},
		{name: "cero", matrix: Matrix{{0, 0}, {0, 0}, {0, 0}}},
		{name: "valores finitos grandes", matrix: Matrix{{1e308}, {1e307}}},
		{name: "cancelación extrema de la proyección", matrix: Matrix{{0, 1.3e308}, {1, 8e307}, {1, -8e307}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			q, r, err := ReducedQR(test.matrix)
			if err != nil {
				t.Fatalf("ReducedQR() devolvió un error: %v", err)
			}
			m, n, k := len(test.matrix), len(test.matrix[0]), min(len(test.matrix), len(test.matrix[0]))
			if len(q) != m || len(q[0]) != k || len(r) != k || len(r[0]) != n {
				t.Fatalf("dimensiones inesperadas Q=%dx%d R=%dx%d", len(q), len(q[0]), len(r), len(r[0]))
			}
			assertMatrixApprox(t, multiply(q, r), test.matrix, 1e-11)
			assertMatrixApprox(t, multiply(transpose(q), q), identity(k), 1e-12)
		})
	}
}

func TestReducedQRRejectsNonFiniteResult(t *testing.T) {
	_, _, err := ReducedQR(Matrix{{math.MaxFloat64}, {math.MaxFloat64}})
	if !errors.Is(err, ErrNumericalRange) {
		t.Fatalf("ReducedQR() devolvió el error %v; se esperaba ErrNumericalRange", err)
	}
}

func multiply(left, right Matrix) Matrix {
	result := make(Matrix, len(left))
	for row := range left {
		result[row] = make([]float64, len(right[0]))
		for column := range right[0] {
			rightColumn := make([]float64, len(right))
			for inner := range right {
				rightColumn[inner] = right[inner][column]
			}
			value, finite := scaleSafeDot(left[row], rightColumn)
			if !finite {
				value = math.NaN()
			}
			result[row][column] = value
		}
	}
	return result
}

func transpose(matrix Matrix) Matrix {
	result := make(Matrix, len(matrix[0]))
	for row := range result {
		result[row] = make([]float64, len(matrix))
		for column := range matrix {
			result[row][column] = matrix[column][row]
		}
	}
	return result
}

func identity(size int) Matrix {
	result := make(Matrix, size)
	for row := range result {
		result[row] = make([]float64, size)
		result[row][row] = 1
	}
	return result
}

func assertMatrixApprox(t *testing.T, got, want Matrix, tolerance float64) {
	t.Helper()
	for row := range want {
		for column := range want[row] {
			if !isFinite(got[row][column]) || !isFinite(want[row][column]) {
				t.Fatalf("matrix[%d][%d] no es finito: se obtuvo %.16g; se esperaba %.16g", row, column, got[row][column], want[row][column])
			}
			scale := math.Max(1, math.Abs(want[row][column]))
			if math.Abs(got[row][column]-want[row][column]) > tolerance*scale {
				t.Fatalf("matrix[%d][%d] = %.16g; se esperaba %.16g", row, column, got[row][column], want[row][column])
			}
		}
	}
}
