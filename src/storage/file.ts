import { chmod, mkdir, readFile, unlink } from 'node:fs/promises';
import { dirname } from 'node:path';
import writeFileAtomic from 'write-file-atomic';

export interface WriteOptions {
  /** File mode applied after atomic rename. Default 0o644. */
  mode?: number;
  /** Parent dir mode if it has to be created. Default 0o700. */
  dirMode?: number;
}

export const ensureDir = async (path: string, mode = 0o700): Promise<void> => {
  await mkdir(path, { recursive: true, mode });
};

export const readJson = async <T>(path: string): Promise<T | null> => {
  try {
    const raw = await readFile(path, 'utf8');
    return JSON.parse(raw) as T;
  } catch (err) {
    if ((err as NodeJS.ErrnoException).code === 'ENOENT') return null;
    throw err;
  }
};

export const writeJsonAtomic = async (
  path: string,
  data: unknown,
  opts: WriteOptions = {},
): Promise<void> => {
  const mode = opts.mode ?? 0o644;
  const dirMode = opts.dirMode ?? 0o700;
  await ensureDir(dirname(path), dirMode);
  const json = `${JSON.stringify(data, null, 2)}\n`;
  await writeFileAtomic(path, json, { mode });
  // Best-effort tighten on platforms where mode in atomic write is honoured;
  // chmod a second time so a 0o600 we asked for actually applies.
  try {
    await chmod(path, mode);
  } catch {
    /* Windows ACLs etc — not a real failure */
  }
};

export const removeFile = async (path: string): Promise<void> => {
  try {
    await unlink(path);
  } catch (err) {
    if ((err as NodeJS.ErrnoException).code !== 'ENOENT') throw err;
  }
};
