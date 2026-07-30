import type { NamedMatrix } from '../../domain/matrix.js';
import type { ServiceConfig } from '../config.js';
import { ProblemError, type ValidationIssue } from './problem.js';

export interface StatisticsRequest {
  readonly matrices: readonly NamedMatrix[];
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function rejectUnknownProperties(
  value: Record<string, unknown>,
  allowed: ReadonlySet<string>,
  base: string,
  errors: ValidationIssue[],
): void {
  for (const property of Object.keys(value)) {
    if (!allowed.has(property)) {
      errors.push({ path: `${base}/${property}`, message: 'no se permite esta propiedad' });
    }
  }
}

export function validateStatisticsRequest(
  body: unknown,
  limits: Pick<ServiceConfig, 'maxMatrices' | 'maxRows' | 'maxColumns' | 'maxTotalElements'>,
): StatisticsRequest {
  const errors: ValidationIssue[] = [];
  if (!isRecord(body)) {
    throw new ProblemError(422, 'Error de validación', 'El cuerpo de la solicitud no es válido', 'about:blank', [
      { path: '/matrices', message: 'debe ser un arreglo no vacío' },
    ]);
  }

  rejectUnknownProperties(body, new Set(['matrices']), '', errors);
  if (!Array.isArray(body['matrices'])) {
    errors.push({ path: '/matrices', message: 'debe ser un arreglo no vacío' });
    throw new ProblemError(422, 'Error de validación', 'El cuerpo de la solicitud no es válido', 'about:blank', errors);
  }

  const rawMatrices = body['matrices'];
  if (rawMatrices.length === 0) {
    errors.push({ path: '/matrices', message: 'debe ser un arreglo no vacío' });
  }
  if (rawMatrices.length > limits.maxMatrices) {
    errors.push({ path: '/matrices', message: `debe contener como máximo ${limits.maxMatrices} matrices` });
  }

  const names = new Set<string>();
  let totalElements = 0;
  const matrices: NamedMatrix[] = [];

  rawMatrices.forEach((rawMatrix, matrixIndex) => {
    const base = `/matrices/${matrixIndex}`;
    if (!isRecord(rawMatrix)) {
      errors.push({ path: base, message: 'debe ser un objeto' });
      return;
    }

    rejectUnknownProperties(rawMatrix, new Set(['name', 'values']), base, errors);

    const name = rawMatrix['name'];
    if (typeof name !== 'string' || name.trim().length === 0 || name.length > 100) {
      errors.push({ path: `${base}/name`, message: 'debe ser una cadena no vacía de como máximo 100 caracteres' });
    } else if (names.has(name)) {
      errors.push({ path: `${base}/name`, message: 'debe ser único' });
    } else {
      names.add(name);
    }

    const values = rawMatrix['values'];
    if (!Array.isArray(values) || values.length === 0) {
      errors.push({ path: `${base}/values`, message: 'debe ser un arreglo bidimensional no vacío' });
      return;
    }
    if (values.length > limits.maxRows) {
      errors.push({ path: `${base}/values`, message: `debe contener como máximo ${limits.maxRows} filas` });
    }

    let columns: number | undefined;
    values.forEach((row, rowIndex) => {
      const rowPath = `${base}/values/${rowIndex}`;
      if (!Array.isArray(row) || row.length === 0) {
        errors.push({ path: rowPath, message: 'debe ser un arreglo no vacío' });
        return;
      }
      columns ??= row.length;
      if (row.length !== columns) {
        errors.push({ path: rowPath, message: `debe contener exactamente ${columns} elementos` });
      }
      if (row.length > limits.maxColumns) {
        errors.push({ path: rowPath, message: `debe contener como máximo ${limits.maxColumns} elementos` });
      }
      totalElements += row.length;
      row.forEach((value, columnIndex) => {
        if (typeof value !== 'number' || !Number.isFinite(value)) {
          errors.push({ path: `${rowPath}/${columnIndex}`, message: 'debe ser un número finito' });
        }
      });
    });

    if (typeof name === 'string' && Array.isArray(values)) {
      matrices.push({ name, values: values as number[][] });
    }
  });

  if (totalElements > limits.maxTotalElements) {
    errors.push({
      path: '/matrices',
      message: `debe contener como máximo ${limits.maxTotalElements} elementos en total`,
    });
  }
  if (errors.length > 0) {
    throw new ProblemError(422, 'Error de validación', 'El cuerpo de la solicitud no es válido', 'about:blank', errors);
  }
  return { matrices };
}
