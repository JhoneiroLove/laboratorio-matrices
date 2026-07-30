export interface NamedMatrix {
  readonly name: string;
  readonly values: readonly (readonly number[])[];
}

export interface MatrixStatistics {
  readonly name: string;
  readonly minimum: number;
  readonly maximum: number;
  readonly sum: number;
  readonly average: number;
  readonly elements: number;
  readonly diagonal: boolean;
}

export interface GlobalStatistics {
  readonly minimum: number;
  readonly maximum: number;
  readonly sum: number;
  readonly average: number;
  readonly elements: number;
}

export interface StatisticsResult {
  readonly matrices: readonly MatrixStatistics[];
  readonly global: GlobalStatistics;
  readonly anyDiagonal: boolean;
}
