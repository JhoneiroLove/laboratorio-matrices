import { readFile } from 'node:fs/promises';
import { importSPKI, jwtVerify } from 'jose';
import type { TokenVerifier } from '../../application/ports/token-verifier.js';

export class JoseTokenVerifier implements TokenVerifier {
  private constructor(
    private readonly publicKey: CryptoKey,
    private readonly issuer: string,
    private readonly audience: string,
    private readonly clockToleranceSeconds: number,
  ) {}

  static async fromPemFile(
    path: string,
    issuer: string,
    audience: string,
    clockToleranceSeconds: number,
  ): Promise<JoseTokenVerifier> {
    const pem = await readFile(path, 'utf8');
    return JoseTokenVerifier.fromPem(pem, issuer, audience, clockToleranceSeconds);
  }

  static async fromPem(
    pem: string,
    issuer: string,
    audience: string,
    clockToleranceSeconds: number,
  ): Promise<JoseTokenVerifier> {
    const publicKey = await importSPKI(pem, 'RS256');
    return new JoseTokenVerifier(publicKey, issuer, audience, clockToleranceSeconds);
  }

  async verify(token: string): Promise<void> {
    await jwtVerify(token, this.publicKey, {
      algorithms: ['RS256'],
      issuer: this.issuer,
      audience: this.audience,
      requiredClaims: ['exp', 'nbf'],
      clockTolerance: this.clockToleranceSeconds,
    });
  }
}
