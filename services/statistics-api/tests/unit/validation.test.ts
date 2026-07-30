import { describe, expect, it } from 'vitest';
import { validateStatisticsRequest } from '../../src/infrastructure/http/validation.js';
import { ProblemError } from '../../src/infrastructure/http/problem.js';

const limits = { maxMatrices: 2, maxRows: 2, maxColumns: 3, maxTotalElements: 5 };

function issues(body: unknown): readonly string[] {
  try {
    validateStatisticsRequest(body, limits);
    return [];
  } catch (error) {
    expect(error).toBeInstanceOf(ProblemError);
    return (error as ProblemError).errors?.map(({ path }) => path) ?? [];
  }
}

describe('validateStatisticsRequest', () => {
  it('acepta matrices rectangulares con nombre y números finitos', () => {
    const input = { matrices: [{ name: 'a', values: [[-1.5, 2], [3, 4.25]] }] };
    expect(validateStatisticsRequest(input, limits)).toEqual(input);
  });

  it('rechaza matrices vacías y no rectangulares', () => {
    expect(issues({ matrices: [] })).toContain('/matrices');
    expect(issues({ matrices: [{ name: 'a', values: [[1, 2], [3]] }] }))
      .toContain('/matrices/0/values/1');
  });

  it('rechaza valores no finitos y nombres duplicados', () => {
    const paths = issues({
      matrices: [
        { name: 'same', values: [[Number.NaN]] },
        { name: 'same', values: [[Number.POSITIVE_INFINITY]] },
      ],
    });
    expect(paths).toContain('/matrices/0/values/0/0');
    expect(paths).toContain('/matrices/1/name');
    expect(paths).toContain('/matrices/1/values/0/0');
  });

  it('aplica límites de filas, columnas, matrices y elementos totales', () => {
    const paths = issues({
      matrices: [
        { name: 'a', values: [[1, 2, 3, 4], [5], [6]] },
        { name: 'b', values: [[1]] },
        { name: 'c', values: [[1]] },
      ],
    });
    expect(paths).toContain('/matrices');
    expect(paths).toContain('/matrices/0/values');
    expect(paths).toContain('/matrices/0/values/0');
  });

  it('rechaza propiedades desconocidas en la raíz y en las matrices', () => {
    const paths = issues({
      matrices: [{ name: 'a', values: [[1]], description: 'unknown' }],
      requestMetadata: {},
    });

    expect(paths).toContain('/requestMetadata');
    expect(paths).toContain('/matrices/0/description');
  });
});
