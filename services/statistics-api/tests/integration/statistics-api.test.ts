import { generateKeyPairSync } from 'node:crypto';
import pino from 'pino';
import request from 'supertest';
import { SignJWT } from 'jose';
import { beforeAll, describe, expect, it } from 'vitest';
import { JoseTokenVerifier } from '../../src/infrastructure/auth/jose-token-verifier.js';
import type { ServiceConfig } from '../../src/infrastructure/config.js';
import { createApp } from '../../src/infrastructure/http/app.js';

const issuer = 'https://issuer.test';
const audience = 'statistics-api';
const { publicKey, privateKey } = generateKeyPairSync('rsa', { modulusLength: 2048 });
const publicPem = publicKey.export({ type: 'spki', format: 'pem' }).toString();

const config: ServiceConfig = {
  port: 3000,
  bodyLimit: '10kb',
  corsOrigins: new Set(['https://client.test']),
  jwtPublicKeyPath: 'unused-in-test',
  jwtIssuer: issuer,
  jwtAudience: audience,
  jwtClockToleranceSeconds: 30,
  matrixEpsilon: 0.001,
  maxMatrices: 10,
  maxRows: 10,
  maxColumns: 10,
  maxTotalElements: 100,
  shutdownTimeoutMs: 1000,
  httpRequestTimeoutMs: 30_000,
  httpHeadersTimeoutMs: 10_000,
  httpKeepAliveTimeoutMs: 5_000,
  logLevel: 'silent',
};

describe('API de Estadísticas', () => {
  let app: ReturnType<typeof createApp>;

  beforeAll(async () => {
    const tokenVerifier = await JoseTokenVerifier.fromPem(
      publicPem,
      issuer,
      audience,
      config.jwtClockToleranceSeconds,
    );
    app = createApp({ config, tokenVerifier, logger: pino({ level: 'silent' }) });
  });

  async function token(overrides: { audience?: string; expiration?: string } = {}): Promise<string> {
    return new SignJWT({ sub: 'test-user' })
      .setProtectedHeader({ alg: 'RS256' })
      .setIssuer(issuer)
      .setAudience(overrides.audience ?? audience)
      .setIssuedAt()
      .setNotBefore('0s')
      .setExpirationTime(overrides.expiration ?? '5m')
      .sign(privateKey);
  }

  it('mantiene públicas las rutas de actividad y disponibilidad', async () => {
    await request(app).get('/health/live').expect(200, { status: 'ok' });
    await request(app).get('/health/ready').expect(200, { status: 'ok' });
  });

  it('devuelve estadísticas para un JWT RS256 válido', async () => {
    const response = await request(app)
      .post('/api/v1/statistics')
      .set('authorization', `Bearer ${await token()}`)
      .set('origin', 'https://client.test')
      .send({ matrices: [{ name: 'diagonal', values: [[1, 0.001], [0, -2.5]] }] })
      .expect(200)
      .expect('access-control-allow-origin', 'https://client.test');

    expect(response.body).toEqual({
      matrices: [{
        name: 'diagonal', minimum: -2.5, maximum: 1, sum: -1.499, average: -0.37475, elements: 4, diagonal: true,
      }],
      global: { minimum: -2.5, maximum: 1, sum: -1.499, average: -0.37475, elements: 4 },
      anyDiagonal: true,
    });
    expect(response.headers['x-request-id']).toBeTypeOf('string');
  });

  it('aplica la tolerancia de reloj y rechaza JWT no válidos con un desafío Bearer', async () => {
    const body = { matrices: [{ name: 'a', values: [[1]] }] };
    await request(app)
      .post('/api/v1/statistics')
      .set('authorization', `Bearer ${await token({ expiration: '-20s' })}`)
      .send(body)
      .expect(200);

    await request(app)
      .post('/api/v1/statistics')
      .send(body)
      .expect(401)
      .expect('www-authenticate', 'Bearer')
      .expect('content-type', /application\/problem\+json/);
    await request(app)
      .post('/api/v1/statistics')
      .set('authorization', `Bearer ${await token({ expiration: '-60s' })}`)
      .send(body)
      .expect(401)
      .expect('www-authenticate', 'Bearer');
    await request(app)
      .post('/api/v1/statistics')
      .set('authorization', `Bearer ${await token({ audience: 'other' })}`)
      .send(body)
      .expect(401)
      .expect('www-authenticate', 'Bearer');
  });

  it('devuelve detalles de problema 422 ante un desbordamiento numérico real', async () => {
    const response = await request(app)
      .post('/api/v1/statistics')
      .set('authorization', `Bearer ${await token()}`)
      .send({ matrices: [{ name: 'overflow', values: [[Number.MAX_VALUE, Number.MAX_VALUE]] }] })
      .expect(422)
      .expect('content-type', /application\/problem\+json/);

    expect(response.body).toMatchObject({
      type: 'urn:interseguro:statistics-api:problem:numerical-range',
      title: 'Rango numérico excedido',
      status: 422,
    });
  });

  it('devuelve errores de validación RFC 9457 y rechaza orígenes no permitidos', async () => {
    const auth = `Bearer ${await token()}`;
    const invalid = await request(app)
      .post('/api/v1/statistics')
      .set('authorization', auth)
      .send({ matrices: [{ name: 'ragged', values: [[1, 2], [3]] }] })
      .expect(422)
      .expect('content-type', /application\/problem\+json/);
    expect(invalid.body.errors[0]).toHaveProperty('path');
    expect(invalid.body).toMatchObject({ title: 'Error de validación', status: 422 });

    await request(app)
      .post('/api/v1/statistics')
      .set('origin', 'https://evil.test')
      .set('authorization', auth)
      .send({ matrices: [{ name: 'a', values: [[1]] }] })
      .expect(403);
  });
});
