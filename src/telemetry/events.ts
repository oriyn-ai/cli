// Typed event names used by telemetry/client. Keep in sync with server-side
// PostHog event filters. Never include secrets or hypothesis text in payloads.
export type CliEvent =
  | { name: 'cli_command_invoked'; props: { command: string; sub?: string } }
  | { name: 'cli_command_succeeded'; props: { command: string; duration_ms: number } }
  | { name: 'cli_command_failed'; props: { command: string; exit_code: number } }
  | { name: 'cli_login_started'; props: { no_browser: boolean } }
  | { name: 'cli_login_succeeded'; props: Record<string, never> }
  | { name: 'cli_login_failed'; props: { reason: string } }
  | { name: 'cli_link_created'; props: { source: 'interactive' | 'flag' } }
  | { name: 'cli_experiment_started'; props: { agents?: number; hypothesis_chars: number } }
  | { name: 'cli_experiment_finished'; props: { verdict: string; duration_ms: number } }
  | { name: 'cli_telemetry_disabled'; props: Record<string, never> };
