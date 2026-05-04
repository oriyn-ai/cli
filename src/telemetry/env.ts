// Vendors whose presence implies CI/automation. When present we auto-skip
// telemetry init unless the user has explicitly opted in. Mirrors the Go
// CLI's CIAutoSkip table.
const CI_VENDORS = [
  'GITHUB_ACTIONS',
  'GITLAB_CI',
  'CIRCLECI',
  'TRAVIS',
  'BUILDKITE',
  'DRONE',
  'BITBUCKET_BUILD_NUMBER',
  'TF_BUILD',
  'JENKINS_URL',
  'TEAMCITY_VERSION',
  'VERCEL',
  'NETLIFY',
  'RAILWAY_ENVIRONMENT',
  'CODEBUILD_BUILD_ARN',
  'SEMAPHORE',
  'APPVEYOR',
  'HEROKU_TEST_RUN_ID',
] as const;

export const ciAutoSkip = (env: NodeJS.ProcessEnv = process.env): boolean => {
  if (env.CI === 'true' || env.CI === '1') return true;
  for (const key of CI_VENDORS) {
    if (env[key]) return true;
  }
  return false;
};
