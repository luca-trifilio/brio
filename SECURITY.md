# Security Policy

## Supported Versions

brio is pre-1.0. Only the **latest release** receives security fixes.

| Version | Supported |
|---------|-----------|
| latest  | ✅        |
| older   | ❌        |

## Reporting a Vulnerability

**Please do not open a public issue for security vulnerabilities.**

Use GitHub's private vulnerability reporting instead:

👉 https://github.com/luca-trifilio/brio/security/advisories/new

This keeps the report confidential until a fix is released.

### What to include

- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

### Response timeline

| Milestone | Target |
|---|---|
| Acknowledgement | within 7 days |
| Status update | within 14 days |
| Fix release | within 30 days (severity-dependent) |
| Public disclosure | after fix is released |

We follow [coordinated disclosure](https://en.wikipedia.org/wiki/Coordinated_vulnerability_disclosure):
the vulnerability will be made public once a patched release is available.

## Scope

In scope: the `brio` binary and its dependencies.  
Out of scope: the Bruno collections you point brio at, your environment files,
or your AWS credentials — brio never writes to disk and only reads what you
explicitly pass as arguments.
