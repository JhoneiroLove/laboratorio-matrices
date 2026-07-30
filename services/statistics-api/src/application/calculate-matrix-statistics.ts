import type { NamedMatrix, StatisticsResult } from '../domain/matrix.js';
import { calculateStatistics } from '../domain/statistics.js';

export class CalculateMatrixStatistics {
  constructor(private readonly epsilon: number) {}

  execute(matrices: readonly NamedMatrix[]): StatisticsResult {
    return calculateStatistics(matrices, this.epsilon);
  }
}
