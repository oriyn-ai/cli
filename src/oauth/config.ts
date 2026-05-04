// Clerk OAuth client for the CLI (public client, PKCE-only).
export const CLERK_OAUTH = {
  clientId: 'GeT6xrbcPZsNblNg',
  authorizeUrl: 'https://clerk.oriyn.ai/oauth/authorize',
  tokenUrl: 'https://clerk.oriyn.ai/oauth/token',
  userInfoUrl: 'https://clerk.oriyn.ai/oauth/userinfo',
  issuer: 'https://clerk.oriyn.ai',
  scopes: ['openid', 'email', 'offline_access', 'profile'],
} as const;

export const CALLBACK_PATH = '/callback';
export const CALLBACK_TIMEOUT_MS = 120_000;
