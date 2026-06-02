import { NotLoggedInError, SessionExpiredError } from '../auth/errors.ts';
import { ExitCode } from '../exit-codes.ts';
import { ApiError, NetworkError, PermissionError } from '../http/errors.ts';
import { ui } from '../output/human.ts';
import { writeJson } from '../output/jsonl.ts';
import { resolveMode } from '../output/mode.ts';
import { captureException } from '../telemetry/sentry.ts';

type SentryCaptureContext = {
  tags?: Record<string, string>;
  extra?: Record<string, unknown>;
};

export interface HandledError {
  exit: ExitCode;
  message: string;
  code: string;
  reportable: boolean;
  context?: SentryCaptureContext;
}

type CliErrorOptions = {
  exit: ExitCode;
  code: string;
  reportable?: boolean;
  context?: SentryCaptureContext;
};

export class CliError extends Error {
  readonly exit: ExitCode;
  readonly code: string;
  readonly reportable: boolean;
  readonly context?: SentryCaptureContext;

  constructor(message: string, options: CliErrorOptions) {
    super(message);
    this.name = 'CliError';
    this.exit = options.exit;
    this.code = options.code;
    this.reportable = options.reportable ?? false;
    this.context = options.context;
  }
}

class ExitSignal extends Error {
  readonly exit: ExitCode;

  constructor(exit: ExitCode) {
    super(`oriyn_exit_${exit}`);
    this.name = 'ExitSignal';
    this.exit = exit;
  }
}

export const isExitSignal = (err: unknown): err is ExitSignal => err instanceof ExitSignal;

const classify = (err: unknown): HandledError => {
  if (err instanceof CliError) {
    return {
      exit: err.exit,
      message: err.message,
      code: err.code,
      reportable: err.reportable,
      context: err.context,
    };
  }
  if (err instanceof PermissionError) {
    return {
      exit: ExitCode.Permission,
      message: err.message,
      code: 'permission_denied',
      reportable: true,
    };
  }
  if (err instanceof ApiError) {
    return {
      exit: ExitCode.Api,
      message: err.message,
      code: `api_${err.status}`,
      reportable: true,
    };
  }
  if (err instanceof NetworkError) {
    return {
      exit: ExitCode.Network,
      message: err.message,
      code: 'network_error',
      reportable: true,
    };
  }
  if (err instanceof NotLoggedInError) {
    return { exit: ExitCode.Auth, message: err.message, code: 'not_logged_in', reportable: false };
  }
  if (err instanceof SessionExpiredError) {
    return {
      exit: ExitCode.Auth,
      message: err.message,
      code: 'session_expired',
      reportable: false,
    };
  }
  if (err instanceof Error) {
    return { exit: ExitCode.Generic, message: err.message, code: 'error', reportable: true };
  }
  return { exit: ExitCode.Generic, message: String(err), code: 'error', reportable: true };
};

export const reportAndExit = (err: unknown): never => {
  if (isExitSignal(err)) {
    throw err;
  }
  const handled = classify(err);
  if (handled.reportable) {
    captureException(err, {
      ...handled.context,
      tags: {
        ...handled.context?.tags,
        'cli.error_code': handled.code,
        'cli.exit_code': String(handled.exit),
      },
    });
  }
  if (resolveMode() === 'jsonl') {
    writeJson({ error: handled.message, code: handled.code, exit: handled.exit }, process.stderr);
  } else {
    process.stderr.write(`${ui.red(ui.cross())} ${handled.message}\n`);
  }
  process.exitCode = handled.exit;
  throw new ExitSignal(handled.exit);
};
