import { ExitCode } from '../exit-codes.ts';

export class ApiError extends Error {
  constructor(
    message: string,
    readonly status: number,
    readonly body: unknown,
  ) {
    super(message);
    this.name = 'ApiError';
  }
  get exitCode(): ExitCode {
    return ExitCode.Api;
  }
}

export class PermissionError extends ApiError {
  constructor(
    message: string,
    status: number,
    body: unknown,
    readonly requiredPermission: string,
  ) {
    super(message, status, body);
    this.name = 'PermissionError';
  }
  override get exitCode(): ExitCode {
    return ExitCode.Permission;
  }
}

export class NetworkError extends Error {
  constructor(message: string, options?: { cause?: unknown }) {
    super(message, options);
    this.name = 'NetworkError';
  }
  get exitCode(): ExitCode {
    return ExitCode.Network;
  }
}
