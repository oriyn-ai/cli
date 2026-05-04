import * as oauth from 'oauth4webapi';
import { CLERK_OAUTH } from './config.ts';

const asAuthServer = (): oauth.AuthorizationServer => ({
  issuer: CLERK_OAUTH.issuer,
  authorization_endpoint: CLERK_OAUTH.authorizeUrl,
  token_endpoint: CLERK_OAUTH.tokenUrl,
  userinfo_endpoint: CLERK_OAUTH.userInfoUrl,
});

const client = (): oauth.Client => ({
  client_id: CLERK_OAUTH.clientId,
  token_endpoint_auth_method: 'none',
});

export interface AuthorizeUrlInput {
  codeChallenge: string;
  state: string;
  redirectUri: string;
}

export const buildAuthorizeUrl = (input: AuthorizeUrlInput): URL => {
  const url = new URL(CLERK_OAUTH.authorizeUrl);
  url.searchParams.set('client_id', CLERK_OAUTH.clientId);
  url.searchParams.set('redirect_uri', input.redirectUri);
  url.searchParams.set('response_type', 'code');
  url.searchParams.set('scope', CLERK_OAUTH.scopes.join(' '));
  url.searchParams.set('state', input.state);
  url.searchParams.set('code_challenge', input.codeChallenge);
  url.searchParams.set('code_challenge_method', 'S256');
  return url;
};

export interface TokenSet {
  accessToken: string;
  refreshToken: string;
  expiresAt: number;
  scope: string | undefined;
}

const computeExpiresAt = (expiresIn: number | undefined): number => {
  const now = Math.floor(Date.now() / 1000);
  return now + (expiresIn ?? 3600);
};

const noClientAuth: oauth.ClientAuth = oauth.None();

export interface ExchangeCodeInput {
  code: string;
  codeVerifier: string;
  redirectUri: string;
}

export const exchangeCode = async (input: ExchangeCodeInput): Promise<TokenSet> => {
  const as = asAuthServer();
  const c = client();
  const response = await oauth.authorizationCodeGrantRequest(
    as,
    c,
    noClientAuth,
    new URLSearchParams({
      grant_type: 'authorization_code',
      code: input.code,
      redirect_uri: input.redirectUri,
      client_id: CLERK_OAUTH.clientId,
      code_verifier: input.codeVerifier,
    }),
    input.redirectUri,
    input.codeVerifier,
  );
  const result = await oauth.processAuthorizationCodeResponse(as, c, response);
  if (!result.refresh_token) {
    throw new Error('OAuth provider did not return a refresh token');
  }
  return {
    accessToken: result.access_token,
    refreshToken: result.refresh_token,
    expiresAt: computeExpiresAt(result.expires_in),
    scope: typeof result.scope === 'string' ? result.scope : undefined,
  };
};

export const refreshTokens = async (refreshToken: string): Promise<TokenSet> => {
  const as = asAuthServer();
  const c = client();
  const response = await oauth.refreshTokenGrantRequest(as, c, noClientAuth, refreshToken);
  const result = await oauth.processRefreshTokenResponse(as, c, response);
  return {
    accessToken: result.access_token,
    // Clerk rotates; if the response omits refresh_token, keep the old one.
    refreshToken: result.refresh_token ?? refreshToken,
    expiresAt: computeExpiresAt(result.expires_in),
    scope: typeof result.scope === 'string' ? result.scope : undefined,
  };
};

export interface UserInfo {
  sub: string;
  email: string | undefined;
}

export const fetchUserInfo = async (accessToken: string): Promise<UserInfo> => {
  const as = asAuthServer();
  const c = client();
  const response = await oauth.userInfoRequest(as, c, accessToken);
  const result = await oauth.processUserInfoResponse(as, c, oauth.skipSubjectCheck, response);
  return {
    sub: result.sub,
    email: typeof result.email === 'string' ? result.email : undefined,
  };
};
