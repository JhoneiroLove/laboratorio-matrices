export interface ServiceConfig {
  readonly port: number;
  readonly bodyLimit: string;
  readonly corsOrigins: ReadonlySet<string>;
  readonly jwtPublicKeyPath: string;
  readonly jwtIssuer: string;
  readonly jwtAudience: string;
  readonly jwtClockToleranceSeconds: number;
  readonly matrixEpsilon: number;
  readonly maxMatrices: number;
  readonly maxRows: number;
  readonly maxColumns: number;
  readonly maxTotalElements: number;
  readonly shutdownTimeoutMs: number;
  readonly httpRequestTimeoutMs: number;
  readonly httpHeadersTimeoutMs: number;
  readonly httpKeepAliveTimeoutMs: number;
  readonly logLevel: string;
}

function required(env: NodeJS.ProcessEnv, name: string): string {
  const value = env[name]?.trim();
  if (!value) throw new Error(`${name} es obligatoria`);
  return value;
}

function positiveInteger(env: NodeJS.ProcessEnv, name: string, fallback: number): number {
  const raw = env[name];
  if (raw === undefined) return fallback;
  const value = Number(raw);
  if (!Number.isSafeInteger(value) || value <= 0) {
    throw new Error(`${name} debe ser un entero positivo`);
  }
  return value;
}

function nonNegativeNumber(env: NodeJS.ProcessEnv, name: string, fallback: number): number {
  const raw = env[name];
  if (raw === undefined) return fallback;
  const value = Number(raw);
  if (!Number.isFinite(value) || value < 0) {
    throw new Error(`${name} debe ser un número finito no negativo`);
  }
  return value;
}

export function loadConfig(env: NodeJS.ProcessEnv = process.env): ServiceConfig {
  const origins = required(env, 'CORS_ORIGINS')
    .split(',')
    .map((origin) => origin.trim())
    .filter(Boolean);

  if (origins.length === 0 || origins.includes('*')) {
    throw new Error('CORS_ORIGINS debe contener orígenes explícitos');
  }

  const httpRequestTimeoutMs = positiveInteger(env, 'HTTP_REQUEST_TIMEOUT_MS', 30_000);
  const httpHeadersTimeoutMs = positiveInteger(env, 'HTTP_HEADERS_TIMEOUT_MS', 10_000);
  if (httpHeadersTimeoutMs > httpRequestTimeoutMs) {
    throw new Error('HTTP_HEADERS_TIMEOUT_MS no debe superar HTTP_REQUEST_TIMEOUT_MS');
  }

  return {
    port: positiveInteger(env, 'PORT', 3000),
    bodyLimit: env['JSON_BODY_LIMIT']?.trim() || '256kb',
    corsOrigins: new Set(origins),
    jwtPublicKeyPath: required(env, 'JWT_PUBLIC_KEY_PATH'),
    jwtIssuer: required(env, 'JWT_ISSUER'),
    jwtAudience: required(env, 'JWT_AUDIENCE'),
    jwtClockToleranceSeconds: nonNegativeNumber(env, 'JWT_CLOCK_TOLERANCE_SECONDS', 30),
    matrixEpsilon: nonNegativeNumber(env, 'MATRIX_EPSILON', 1e-10),
    maxMatrices: positiveInteger(env, 'MAX_MATRICES', 100),
    maxRows: positiveInteger(env, 'MAX_MATRIX_ROWS', 1000),
    maxColumns: positiveInteger(env, 'MAX_MATRIX_COLUMNS', 1000),
    maxTotalElements: positiveInteger(env, 'MAX_TOTAL_ELEMENTS', 1_000_000),
    shutdownTimeoutMs: positiveInteger(env, 'SHUTDOWN_TIMEOUT_MS', 10_000),
    httpRequestTimeoutMs,
    httpHeadersTimeoutMs,
    httpKeepAliveTimeoutMs: positiveInteger(env, 'HTTP_KEEP_ALIVE_TIMEOUT_MS', 5_000),
    logLevel: env['LOG_LEVEL']?.trim() || 'info',
  };
}
