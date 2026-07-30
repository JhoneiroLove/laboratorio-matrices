export interface TokenVerifier {
  verify(token: string): Promise<void>;
}
