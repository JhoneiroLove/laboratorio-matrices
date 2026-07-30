import { createServer } from 'node:http';
import pino from 'pino';
import { JoseTokenVerifier } from './infrastructure/auth/jose-token-verifier.js';
import { loadConfig } from './infrastructure/config.js';
import { createApp } from './infrastructure/http/app.js';

const config = loadConfig();
const logger = pino({ level: config.logLevel });
const tokenVerifier = await JoseTokenVerifier.fromPemFile(
  config.jwtPublicKeyPath,
  config.jwtIssuer,
  config.jwtAudience,
  config.jwtClockToleranceSeconds,
);
const app = createApp({ config, tokenVerifier, logger });
const server = createServer(app);
server.requestTimeout = config.httpRequestTimeoutMs;
server.headersTimeout = config.httpHeadersTimeoutMs;
server.keepAliveTimeout = config.httpKeepAliveTimeoutMs;

server.listen(config.port, () => {
  logger.info({ port: config.port }, 'statistics-api está escuchando');
});

let shuttingDown = false;
function shutdown(signal: NodeJS.Signals): void {
  if (shuttingDown) return;
  shuttingDown = true;
  logger.info({ signal }, 'se inició el cierre ordenado');

  const timeout = setTimeout(() => {
    logger.error('se agotó el tiempo del cierre ordenado');
    server.closeAllConnections();
  }, config.shutdownTimeoutMs);
  timeout.unref();

  server.close((error) => {
    clearTimeout(timeout);
    if (error) {
      logger.error({ err: error }, 'falló el cierre del servidor');
      process.exitCode = 1;
      return;
    }
    logger.info('se completó el cierre ordenado');
  });
}

process.once('SIGTERM', shutdown);
process.once('SIGINT', shutdown);
