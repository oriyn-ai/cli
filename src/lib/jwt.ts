// Decode-only JWT helper. We don't verify signatures here — the access token
// is issued by Clerk to us, and we only read claims for display + best-effort
// org context. Anything load-bearing is verified server-side.
export interface JwtClaims {
  sub?: string;
  email?: string;
  org_id?: string;
  org_slug?: string;
  org_role?: string;
}

export const decodeJwtPayload = (token: string): JwtClaims => {
  const parts = token.split('.');
  if (parts.length !== 3) return {};
  try {
    const base64url = parts[1] ?? '';
    const base64 = base64url.replace(/-/g, '+').replace(/_/g, '/');
    const padded = base64 + '='.repeat((4 - (base64.length % 4)) % 4);
    const decoded = Buffer.from(padded, 'base64').toString('utf8');
    return JSON.parse(decoded) as JwtClaims;
  } catch {
    return {};
  }
};
