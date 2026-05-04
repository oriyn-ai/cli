import { ExitCode } from '../exit-codes.ts';
import { type ResolveOptions, resolveProduct } from '../link/resolver.ts';

export class NotLinkedError extends Error {
  override readonly name = 'NotLinkedError';
  readonly exitCode = ExitCode.Generic;
  constructor(
    message = 'No product linked here. Run `oriyn link` from your project, or pass --product <id>.',
  ) {
    super(message);
  }
}

export const requireProduct = async (
  opts: ResolveOptions = {},
): Promise<{ productId: string; orgId: string | undefined; source: string }> => {
  const resolved = await resolveProduct(opts);
  if (!resolved) throw new NotLinkedError();
  return resolved;
};
