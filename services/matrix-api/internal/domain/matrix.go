package domain

import (
	"errors"
	"fmt"
	"math"
)

type Matrix [][]float64

type Limits struct {
	MaxRows     int
	MaxColumns  int
	MaxElements int
}

type Dimensions struct {
	Rows    int
	Columns int
}

var ErrInvalidMatrix = errors.New("matriz no válida")
var ErrNumericalRange = errors.New("la matriz supera el rango numérico")

func Validate(matrix Matrix, limits Limits) (Dimensions, error) {
	if limits.MaxRows <= 0 || limits.MaxColumns <= 0 || limits.MaxElements <= 0 {
		return Dimensions{}, fmt.Errorf("%w: los límites deben ser positivos", ErrInvalidMatrix)
	}
	if len(matrix) == 0 {
		return Dimensions{}, fmt.Errorf("%w: se requiere al menos una fila", ErrInvalidMatrix)
	}
	if len(matrix) > limits.MaxRows {
		return Dimensions{}, fmt.Errorf("%w: la cantidad de filas supera el límite de %d", ErrInvalidMatrix, limits.MaxRows)
	}

	columns := len(matrix[0])
	if columns == 0 {
		return Dimensions{}, fmt.Errorf("%w: se requiere al menos una columna", ErrInvalidMatrix)
	}
	if columns > limits.MaxColumns {
		return Dimensions{}, fmt.Errorf("%w: la cantidad de columnas supera el límite de %d", ErrInvalidMatrix, limits.MaxColumns)
	}
	if len(matrix) > limits.MaxElements/columns {
		return Dimensions{}, fmt.Errorf("%w: la cantidad de elementos supera el límite de %d", ErrInvalidMatrix, limits.MaxElements)
	}

	for rowIndex, row := range matrix {
		if len(row) != columns {
			return Dimensions{}, fmt.Errorf("%w: la fila %d tiene %d columnas; se esperaban %d", ErrInvalidMatrix, rowIndex, len(row), columns)
		}
		for columnIndex, value := range row {
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return Dimensions{}, fmt.Errorf("%w: el elemento [%d][%d] debe ser finito", ErrInvalidMatrix, rowIndex, columnIndex)
			}
		}
	}

	return Dimensions{Rows: len(matrix), Columns: columns}, nil
}

func RotateClockwise(matrix Matrix) Matrix {
	rows, columns := len(matrix), len(matrix[0])
	rotated := make(Matrix, columns)
	for column := 0; column < columns; column++ {
		rotated[column] = make([]float64, rows)
		for row := 0; row < rows; row++ {
			rotated[column][rows-1-row] = matrix[row][column]
		}
	}
	return rotated
}

// ReducedQR calcula A = QR mediante reflexiones de Householder. Q tiene dimensiones
// m x min(m,n) y R tiene dimensiones min(m,n) x n.
func ReducedQR(matrix Matrix) (Matrix, Matrix, error) {
	m, n := len(matrix), len(matrix[0])
	k := min(m, n)
	rWork := clone(matrix)
	reflectors := make([][]float64, k)
	betas := make([]float64, k)

	for column := 0; column < k; column++ {
		length := m - column
		v := make([]float64, length)
		for row := 0; row < length; row++ {
			v[row] = rWork[column+row][column]
		}

		norm := stableNorm(v)
		if math.IsInf(norm, 0) || math.IsNaN(norm) {
			return nil, nil, fmt.Errorf("%w: la norma de la columna %d no es finita", ErrNumericalRange, column)
		}
		if norm == 0 {
			reflectors[column] = v
			continue
		}
		for row := range v {
			v[row] /= norm
		}
		alphaUnit := -math.Copysign(1, v[0])
		v[0] -= alphaUnit
		denominator, finite := scaleSafeDot(v, v)
		if !finite {
			return nil, nil, fmt.Errorf("%w: la norma del reflector no es finita", ErrNumericalRange)
		}
		if denominator == 0 {
			reflectors[column] = v
			continue
		}
		beta := 2 / denominator
		reflectors[column], betas[column] = v, beta

		// La columna pivote tiene un resultado exacto conocido. Asignarlo directamente
		// evita el desbordamiento de una reflexión innecesaria con valores muy grandes.
		for targetColumn := column + 1; targetColumn < n; targetColumn++ {
			projection, finite := scaledProjection(v, beta, rWork, column, targetColumn)
			if !finite {
				return nil, nil, fmt.Errorf("%w: la proyección no es finita", ErrNumericalRange)
			}
			for row := 0; row < length; row++ {
				rWork[column+row][targetColumn] -= projection * v[row]
				if !isFinite(rWork[column+row][targetColumn]) {
					return nil, nil, fmt.Errorf("%w: el resultado QR no es finito", ErrNumericalRange)
				}
			}
		}
		rWork[column][column] = alphaUnit * norm
		for row := column + 1; row < m; row++ {
			rWork[row][column] = 0
		}
	}

	q := make(Matrix, m)
	for row := range q {
		q[row] = make([]float64, k)
		if row < k {
			q[row][row] = 1
		}
	}
	for column := k - 1; column >= 0; column-- {
		v, beta := reflectors[column], betas[column]
		if beta == 0 {
			continue
		}
		for targetColumn := 0; targetColumn < k; targetColumn++ {
			projection, finite := scaledProjection(v, beta, q, column, targetColumn)
			if !finite {
				return nil, nil, fmt.Errorf("%w: la proyección de Q no es finita", ErrNumericalRange)
			}
			for row := range v {
				q[column+row][targetColumn] -= projection * v[row]
				if !isFinite(q[column+row][targetColumn]) {
					return nil, nil, fmt.Errorf("%w: el resultado de Q no es finito", ErrNumericalRange)
				}
			}
		}
	}

	r := make(Matrix, k)
	for row := 0; row < k; row++ {
		r[row] = append([]float64(nil), rWork[row]...)
		for column := 0; column < row && column < n; column++ {
			r[row][column] = 0
		}
	}
	if !matrixIsFinite(q) || !matrixIsFinite(r) {
		return nil, nil, fmt.Errorf("%w: el resultado QR no es finito", ErrNumericalRange)
	}
	return q, r, nil
}

func clone(matrix Matrix) Matrix {
	result := make(Matrix, len(matrix))
	for row := range matrix {
		result[row] = append([]float64(nil), matrix[row]...)
	}
	return result
}

func stableNorm(values []float64) float64 {
	scale, sumSquares := 0.0, 1.0
	for _, value := range values {
		absolute := math.Abs(value)
		if absolute == 0 {
			continue
		}
		if scale < absolute {
			ratio := scale / absolute
			sumSquares = 1 + sumSquares*ratio*ratio
			scale = absolute
		} else {
			ratio := absolute / scale
			sumSquares += ratio * ratio
		}
	}
	if scale == 0 {
		return 0
	}
	return scale * math.Sqrt(sumSquares)
}

// scaleSafeDot escala el vector derecho antes de la acumulación compensada. Así
// evita que sumas parciales del mismo signo desborden antes de una cancelación posterior.
func scaleSafeDot(left, right []float64) (float64, bool) {
	scale := 0.0
	for _, value := range right {
		scale = math.Max(scale, math.Abs(value))
	}
	if scale == 0 {
		return 0, true
	}

	sum, correction := 0.0, 0.0
	for index := range left {
		term := left[index] * (right[index] / scale)
		next := sum + term
		if math.Abs(sum) >= math.Abs(term) {
			correction += (sum - next) + term
		} else {
			correction += (term - next) + sum
		}
		sum = next
	}
	result := (sum + correction) * scale
	return result, isFinite(result)
}

func scaledProjection(v []float64, beta float64, matrix Matrix, rowOffset, column int) (float64, bool) {
	scale := 0.0
	for row := range v {
		scale = math.Max(scale, math.Abs(matrix[rowOffset+row][column]))
	}
	if scale == 0 {
		return 0, true
	}

	sum, correction := 0.0, 0.0
	for row := range v {
		term := (beta * v[row]) * (matrix[rowOffset+row][column] / scale)
		next := sum + term
		if math.Abs(sum) >= math.Abs(term) {
			correction += (sum - next) + term
		} else {
			correction += (term - next) + sum
		}
		sum = next
	}
	result := (sum + correction) * scale
	return result, isFinite(result)
}

func matrixIsFinite(matrix Matrix) bool {
	for _, row := range matrix {
		for _, value := range row {
			if !isFinite(value) {
				return false
			}
		}
	}
	return true
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
