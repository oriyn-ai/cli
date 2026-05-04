import { NotLoggedInError, SessionExpiredError } from '../auth/errors.ts';
import { ExitCode } from '../exit-codes.ts';
import { ApiError, NetworkError, PermissionError } from '../http/errors.ts';
import { ui } from '../output/human.ts';
import { writeJson } from '../output/jsonl.ts';
import { resolveMode } from '../output/mode.ts';
import { captureException } from '../telemetry/sentry.ts';

export interface HandledError {
  exit: ExitCode;
  message: string;
  code: string;
}

const classify = (err: unknown): HandledError => {
  if (err instanceof PermissionError) {
    return { exit: ExitCode.Permission, message: err.message, code: 'permission_denied' };
  }
  if (err instanceof ApiError) {
    return { exit: ExitCode.Api, message: err.message, code: `api_${err.status}` };
  }
  if (err instanceof NetworkError) {
    return { exit: ExitCode.Network, message: err.message, code: 'network_error' };
  }
  if (err instanceof NotLoggedInError) {
    return { exit: ExitCode.Auth, message: err.message, code: 'not_logged_in' };
  }
  if (err instanceof SessionExpiredError) {
    return { exit: ExitCode.Auth, message: err.message, code: 'session_expired' };
  }
  if (err instanceof Error) {
    return { exit: ExitCode.Generic, message: err.message, code: 'error' };
  }
  return { exit: ExitCode.Generic, message: String(err), code: 'error' };
};

export const reportAndExit = (err: unknown): never => {
  const handled = classify(err);
  if (handled.exit >= ExitCode.Api) {
    captureException(err);
  }
  if (resolveMode() === 'jsonl') {
    writeJson({ error: handled.message, code: handled.code, exit: handled.exit }, process.stderr);
  } else {
    process.stderr.write(`${ui.red(ui.cross())} ${handled.message}\n`);
  }
  process.exit(handled.exit);
};
