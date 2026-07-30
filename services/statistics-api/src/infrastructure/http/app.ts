import { randomUUID } from 'node:crypto';
import cors from 'cors';
import express, { type ErrorRequestHandler, type NextFunction, type Request, type Response } from 'express';
import helmet from 'helmet';
import type { Logger } from 'pino';
import { pinoHttp } from 'pino-http';
import { CalculateMatrixStatistics } from '../../application/calculate-matrix-statistics.js';
import type { TokenVerifier } from '../../application/ports/token-verifier.js';
import { NumericalRangeError } from '../../domain/errors.js';
import type { ServiceConfig } from '../config.js';
import { ProblemError, sendProblem } from './problem.js';
import { validateStatisticsRequest } from './validation.js';

export interface AppDependencies {
  readonly config: ServiceConfig;
  readonly tokenVerifier: TokenVerifier;
  readonly logger: Logger;
  readonly isReady?: () => boolean;
}

function requestId(req: Request, res: Response, next: NextFunction): void {
  const supplied = req.get('x-request-id');
  const id = supplied && /^[A-Za-z0-9._:-]{1,128}$/.test(supplied) ? supplied : randomUUID();
  res.locals['requestId'] = id;
  res.setHeader('x-request-id', id);
  next();
}

function bearerToken(req: Request): string {
  const authorization = req.get('authorization');
  const match = authorization?.match(/^Bearer ([^\s]+)$/i);
  if (!match?.[1]) {
    throw new ProblemError(401, 'No autorizado', 'Se requiere un token Bearer');
  }
  return match[1];
}

function parserProblem(error: unknown): ProblemError | undefined {
  if (!(error instanceof Error)) return undefined;
  const type = 'type' in error ? error.type : undefined;
  if (type === 'entity.too.large') {
    return new ProblemError(413, 'Contenido demasiado grande', 'El cuerpo JSON supera el límite configurado');
  }
  if (error instanceof SyntaxError && 'body' in error) {
    return new ProblemError(400, 'Solicitud incorrecta', 'El cuerpo JSON está mal formado');
  }
  return undefined;
}

export function createApp({ config, tokenVerifier, logger, isReady = () => true }: AppDependencies) {
  const app = express();
  const calculate = new CalculateMatrixStatistics(config.matrixEpsilon);

  app.disable('x-powered-by');
  app.use(requestId);
  app.use(
    pinoHttp({
      logger,
      redact: ['req.headers.authorization'],
      autoLogging: {
        ignore: (req) => req.url === '/health/live' || req.url === '/health/ready',
      },
      customProps: (_req, res: Response<unknown, { requestId?: string }>) => ({
        requestId: res.locals.requestId,
      }),
    }),
  );
  app.use(helmet());
  app.use(
    cors({
      origin(origin, callback) {
        if (!origin || config.corsOrigins.has(origin)) return callback(null, true);
        return callback(new ProblemError(403, 'Acceso prohibido', 'El origen no está permitido'));
      },
      methods: ['GET', 'POST'],
      allowedHeaders: ['authorization', 'content-type', 'x-request-id'],
      exposedHeaders: ['x-request-id'],
      maxAge: 600,
    }),
  );
  app.use(express.json({ limit: config.bodyLimit, strict: true, type: 'application/json' }));

  app.get('/health/live', (_req, res) => {
    res.json({ status: 'ok' });
  });
  app.get('/health/ready', (req, res) => {
    if (!isReady()) {
      sendProblem(req, res, new ProblemError(503, 'Servicio no disponible', 'El servicio no está listo'));
      return;
    }
    res.json({ status: 'ok' });
  });

  app.post('/api/v1/statistics', async (req, res) => {
    if (!req.is('application/json')) {
      throw new ProblemError(415, 'Tipo de contenido no compatible', 'Content-Type debe ser application/json');
    }
    try {
      await tokenVerifier.verify(bearerToken(req));
    } catch (error) {
      if (error instanceof ProblemError) throw error;
      throw new ProblemError(401, 'No autorizado', 'El token Bearer no es válido o está vencido');
    }

    const input = validateStatisticsRequest(req.body, config);
    res.json(calculate.execute(input.matrices));
  });

  app.use((req, res) => {
    sendProblem(req, res, new ProblemError(404, 'No encontrado', 'No se encontró el recurso'));
  });

  const errorHandler: ErrorRequestHandler = (error, req, res, next) => {
    void next;
    const problem = error instanceof ProblemError
      ? error
      : error instanceof NumericalRangeError
        ? new ProblemError(
          422,
          'Rango numérico excedido',
          error.message,
          'urn:interseguro:statistics-api:problem:numerical-range',
        )
        : parserProblem(error) ?? new ProblemError(500, 'Error interno del servidor', 'Ocurrió un error inesperado');

    if (problem.status === 401) res.setHeader('www-authenticate', 'Bearer');
    if (problem.status >= 500) req.log.error({ err: error }, 'la solicitud falló');
    else req.log.warn({ status: problem.status, err: error }, 'la solicitud fue rechazada');
    sendProblem(req, res, problem);
  };
  app.use(errorHandler);

  return app;
}
