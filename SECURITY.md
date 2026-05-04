# Security policy

## Supported versions

We support the latest minor version of `oriyn` published to npm. Security
fixes are released as patch versions and announced in
[CHANGELOG.md](./CHANGELOG.md).

## Reporting a vulnerability

**Please do not open a public GitHub issue for security reports.**

Email **[shivam@oriyn.ai](mailto:shivam@oriyn.ai)** with:

- A description of the issue and its impact
- Steps to reproduce (a minimal repo or transcript is ideal)
- The version of the CLI (`oriyn --version`) and your OS

We will acknowledge within **2 business days** and aim to ship a fix or
mitigation within **14 days** for high-severity reports. We are happy to
credit reporters in the release notes unless you prefer to remain anonymous.

## Scope

In scope:

- The `oriyn` CLI binary and its npm package
- The OAuth 2.1 + PKCE login flow (loopback callback handling, token storage,
  redact paths)
- The `install.sh` installer
- Release artifacts (npm, GitHub Releases binaries, checksums)

Out of scope:

- The Oriyn web app and API — report those to the same address; they are
  tracked separately
- Bugs in upstream dependencies (report to the upstream project; we'll bump)
- Issues that require a malicious local user with write access to your home
  directory

## Disclosure

We follow coordinated disclosure. Please give us a reasonable window to ship
a fix before going public. Once a fix is released, we'll credit you (with
permission) in the release notes.
