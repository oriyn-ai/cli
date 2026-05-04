import * as oauth from 'oauth4webapi';

export interface Pkce {
  codeVerifier: string;
  codeChallenge: string;
}

export const generatePkce = async (): Promise<Pkce> => {
  const codeVerifier = oauth.generateRandomCodeVerifier();
  const codeChallenge = await oauth.calculatePKCECodeChallenge(codeVerifier);
  return { codeVerifier, codeChallenge };
};

export const generateState = (): string => oauth.generateRandomState();
