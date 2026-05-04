export const ExitCode = {
  Ok: 0,
  Generic: 1,
  Api: 2,
  Auth: 3,
  Network: 4,
  Permission: 5,
} as const;

export type ExitCode = (typeof ExitCode)[keyof typeof ExitCode];
