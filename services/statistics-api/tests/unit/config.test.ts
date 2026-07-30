import { describe, expect, it } from 'vitest';
import { loadConfig } from '../../src/infrastructure/config.js';

const requiredEnv: NodeJS.ProcessEnv = {
  CORS_ORIGINS: 'https://client.test',
  JWT_PUBLIC_KEY_PATH: '/keys/public.pem',
  JWT_ISSUER: 'https://issuer.test',
  JWT_AUDIENCE: 'statistics-api',
};

describe('loadConfig', () => {
  it('utiliza los valores predeterminados de tolerancia de reloj y tiempos de espera HTTP', () => {
    const config = loadConfig(requiredEnv);

    expect(config.jwtClockToleranceSeconds).toBe(30);
    expect(config.httpRequestTimeoutMs).toBe(30_000);
    expect(config.httpHeadersTimeoutMs).toBe(10_000);
    expect(config.httpKeepAliveTimeoutMs).toBe(5_000);
  });

  it('acepta una tolerancia de reloj JWT finita y no negativa', () => {
    expect(loadConfig({ ...requiredEnv, JWT_CLOCK_TOLERANCE_SECONDS: '2.5' })
      .jwtClockToleranceSeconds).toBe(2.5);
  });

  it('rechaza valores no válidos de tolerancia de reloj y tiempos de espera HTTP', () => {
    expect(() => loadConfig({ ...requiredEnv, JWT_CLOCK_TOLERANCE_SECONDS: '-1' }))
      .toThrow('JWT_CLOCK_TOLERANCE_SECONDS');
    expect(() => loadConfig({
      ...requiredEnv,
      HTTP_REQUEST_TIMEOUT_MS: '1000',
      HTTP_HEADERS_TIMEOUT_MS: '1001',
    })).toThrow('HTTP_HEADERS_TIMEOUT_MS');
    expect(() => loadConfig({ ...requiredEnv, HTTP_KEEP_ALIVE_TIMEOUT_MS: '0' }))
      .toThrow('HTTP_KEEP_ALIVE_TIMEOUT_MS');
  });
});
