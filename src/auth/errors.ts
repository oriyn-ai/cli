import { ExitCode } from '../exit-codes.ts';

export class NotLoggedInError extends Error {
  override readonly name = 'NotLoggedInError';
  readonly exitCode = ExitCode.Auth;
  constructor(message = 'Not logged in. Run `oriyn auth login`.') {
    super(message);
  }
}

export class SessionExpiredError extends Error {
  override readonly name = 'SessionExpiredError';
  readonly exitCode = ExitCode.Auth;
  constructor(message = 'Session expired. Run `oriyn auth login` again.') {
    super(message);
  }
}
