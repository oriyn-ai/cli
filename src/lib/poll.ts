export interface PollOptions<T> {
  fn: () => Promise<T>;
  done: (value: T) => boolean;
  /** Initial wait between attempts (ms). Default 2000. */
  initialDelayMs?: number;
  /** Multiplier for backoff. Default 1.25. */
  factor?: number;
  /** Cap on per-attempt delay. Default 10_000. */
  maxDelayMs?: number;
  /** Total timeout. Default 600_000 (10 min). */
  timeoutMs?: number;
  /** Called between attempts. */
  onTick?: (value: T, elapsedMs: number) => void;
}

export const poll = async <T>(opts: PollOptions<T>): Promise<T> => {
  const start = Date.now();
  const initial = opts.initialDelayMs ?? 2000;
  const factor = opts.factor ?? 1.25;
  const max = opts.maxDelayMs ?? 10_000;
  const timeout = opts.timeoutMs ?? 600_000;

  let delay = initial;
  let last: T = await opts.fn();
  if (opts.done(last)) return last;
  while (true) {
    const elapsed = Date.now() - start;
    if (elapsed >= timeout) throw new Error('Operation timed out');
    await new Promise((r) => setTimeout(r, Math.min(delay, max)));
    last = await opts.fn();
    opts.onTick?.(last, Date.now() - start);
    if (opts.done(last)) return last;
    delay = Math.min(delay * factor, max);
  }
};
