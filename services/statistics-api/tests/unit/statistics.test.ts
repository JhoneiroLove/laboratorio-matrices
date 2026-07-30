import { describe, expect, it } from 'vitest';
import { calculateStatistics } from '../../src/domain/statistics.js';
import { NumericalRangeError } from '../../src/domain/errors.js';

describe('calculateStatistics', () => {
  it('calcula estadísticas por matriz y globales para valores negativos y decimales', () => {
    const result = calculateStatistics(
      [
        { name: 'negative', values: [[-4, -1], [-2, -3]] },
        { name: 'decimal', values: [[0.5, 1.25, 2.25]] },
      ],
      1e-10,
    );

    expect(result.matrices[0]).toEqual({
      name: 'negative', minimum: -4, maximum: -1, sum: -10, average: -2.5, elements: 4, diagonal: false,
    });
    expect(result.matrices[1]).toEqual({
      name: 'decimal', minimum: 0.5, maximum: 2.25, sum: 4, average: 4 / 3, elements: 3, diagonal: false,
    });
    expect(result.global).toEqual({
      minimum: -4, maximum: 2.25, sum: -6, average: -6 / 7, elements: 7,
    });
    expect(result.anyDiagonal).toBe(false);
  });

  it('utiliza suma compensada', () => {
    const result = calculateStatistics(
      [{ name: 'cancellation', values: [[1e16, 1, -1e16]] }],
      0,
    );

    expect(result.matrices[0]?.sum).toBe(1);
    expect(result.global.sum).toBe(1);
  });

  it('evita el desbordamiento intermedio cuando la cancelación con MAX_VALUE es representable', () => {
    const result = calculateStatistics(
      [{ name: 'max-cancellation', values: [[Number.MAX_VALUE, Number.MAX_VALUE, -Number.MAX_VALUE]] }],
      0,
    );

    expect(result.matrices[0]?.sum).toBe(Number.MAX_VALUE);
    expect(result.global.sum).toBe(Number.MAX_VALUE);
  });

  it('lanza un error de rango numérico ante un desbordamiento real', () => {
    expect(() => calculateStatistics(
      [{ name: 'overflow', values: [[Number.MAX_VALUE, Number.MAX_VALUE]] }],
      0,
    )).toThrowError(NumericalRangeError);
  });

  it('marca como diagonales solo las matrices cuadradas cuyos valores externos cumplen epsilon', () => {
    const result = calculateStatistics(
      [
        { name: 'within', values: [[2, 0.001], [-0.001, 4]] },
        { name: 'outside', values: [[2, 0.0011], [0, 4]] },
        { name: 'non-square', values: [[1, 0, 0], [0, 2, 0]] },
      ],
      0.001,
    );

    expect(result.matrices.map(({ diagonal }) => diagonal)).toEqual([true, false, false]);
    expect(result.anyDiagonal).toBe(true);
  });
});
