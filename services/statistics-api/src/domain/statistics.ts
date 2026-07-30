import type {
  GlobalStatistics,
  MatrixStatistics,
  NamedMatrix,
  StatisticsResult,
} from './matrix.js';
import { NumericalRangeError } from './errors.js';

class CompensatedSum {
  private sum = 0;
  private correction = 0;

  add(value: number): void {
    const next = this.sum + value;
    if (Math.abs(this.sum) >= Math.abs(value)) {
      this.correction += this.sum - next + value;
    } else {
      this.correction += value - next + this.sum;
    }
    this.sum = next;
  }

  value(): number {
    return this.sum + this.correction;
  }
}

type ValueVisitor = (consume: (value: number) => void) => void;

function scaledCompensatedSum(visitValues: ValueVisitor): number {
  let scale = 0;
  visitValues((value) => {
    scale = Math.max(scale, Math.abs(value));
  });
  if (scale === 0) return 0;

  const normalized = new CompensatedSum();
  visitValues((value) => normalized.add(value / scale));
  return normalized.value() * scale;
}

function assertFinite(
  scope: string,
  values: Readonly<Record<'minimum' | 'maximum' | 'sum' | 'average', number>>,
): void {
  for (const field of ['minimum', 'maximum', 'sum', 'average'] as const) {
    if (!Number.isFinite(values[field])) throw new NumericalRangeError(scope, field);
  }
}

function isDiagonal(values: NamedMatrix['values'], epsilon: number): boolean {
  if (values.length !== values[0]?.length) return false;

  for (let row = 0; row < values.length; row += 1) {
    for (let column = 0; column < values.length; column += 1) {
      if (row !== column && Math.abs(values[row]![column]!) > epsilon) return false;
    }
  }
  return true;
}

export function calculateStatistics(
  matrices: readonly NamedMatrix[],
  epsilon: number,
): StatisticsResult {
  let globalMinimum = Number.POSITIVE_INFINITY;
  let globalMaximum = Number.NEGATIVE_INFINITY;
  let globalElements = 0;

  const perMatrix: MatrixStatistics[] = matrices.map(({ name, values }) => {
    let minimum = Number.POSITIVE_INFINITY;
    let maximum = Number.NEGATIVE_INFINITY;
    let elements = 0;

    for (const row of values) {
      for (const value of row) {
        minimum = Math.min(minimum, value);
        maximum = Math.max(maximum, value);
        elements += 1;
        globalMinimum = Math.min(globalMinimum, value);
        globalMaximum = Math.max(globalMaximum, value);
        globalElements += 1;
      }
    }

    const total = scaledCompensatedSum((consume) => {
      for (const row of values) for (const value of row) consume(value);
    });
    const average = total / elements;
    assertFinite(`la matriz "${name}"`, { minimum, maximum, sum: total, average });

    return {
      name,
      minimum,
      maximum,
      sum: total,
      average,
      elements,
      diagonal: isDiagonal(values, epsilon),
    };
  });

  const total = scaledCompensatedSum((consume) => {
    for (const { values } of matrices) {
      for (const row of values) for (const value of row) consume(value);
    }
  });
  const average = total / globalElements;
  assertFinite('las estadísticas globales', {
    minimum: globalMinimum,
    maximum: globalMaximum,
    sum: total,
    average,
  });
  const global: GlobalStatistics = {
    minimum: globalMinimum,
    maximum: globalMaximum,
    sum: total,
    average,
    elements: globalElements,
  };

  return {
    matrices: perMatrix,
    global,
    anyDiagonal: perMatrix.some(({ diagonal }) => diagonal),
  };
}
