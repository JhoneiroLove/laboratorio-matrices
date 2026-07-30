import type { Request, Response } from 'express';

export interface ValidationIssue {
  readonly path: string;
  readonly message: string;
}

export class ProblemError extends Error {
  constructor(
    readonly status: number,
    readonly title: string,
    readonly detail: string,
    readonly type = 'about:blank',
    readonly errors?: readonly ValidationIssue[],
  ) {
    super(detail);
  }
}

export function sendProblem(req: Request, res: Response, problem: ProblemError): void {
  const body: Record<string, unknown> = {
    type: problem.type,
    title: problem.title,
    status: problem.status,
    detail: problem.detail,
    instance: req.originalUrl,
    requestId: res.locals['requestId'],
  };
  if (problem.errors) body['errors'] = problem.errors;

  res.status(problem.status).type('application/problem+json').send(body);
}
