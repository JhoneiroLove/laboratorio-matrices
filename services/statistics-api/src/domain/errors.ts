export class NumericalRangeError extends Error {
  constructor(
    readonly scope: string,
    readonly field: 'minimum' | 'maximum' | 'sum' | 'average',
  ) {
    const labels = {
      minimum: 'mínimo',
      maximum: 'máximo',
      sum: 'suma',
      average: 'promedio',
    } as const;
    super(`El valor de ${labels[field]} de ${scope} excede el rango numérico admitido`);
    this.name = 'NumericalRangeError';
  }
}
