# ADR-0003: Auth management must be pluggable, not AWS-only

---
id: ADR-0003
title: Auth management must be pluggable, not AWS-only
status: Accepted
date: 2026-05-08
---

## Context

brio currently hard-wires AWS SigV4 as the only auth mechanism in
internal/httpx. Users working with APIs protected by Bearer tokens,
Basic auth, API keys, or custom credential-refresh flows (OAuth2, SSO)
have no path to configure auth without code changes. The canonical
model introduces AuthBlock with multiple auth modes, but the executor
only acts on AwsV4.

## Decision

Auth must be implemented as a pluggable Authenticator interface in
internal/httpx (or internal/auth). Built-in authenticators: Bearer,
Basic, API key (header or query param), AWS SigV4 (existing), no-op.
A hook-based authenticator delegates to the existing credential-refresh
hook system (internal/hooks), enabling OAuth2 and custom SSO without
new built-ins. Auth is configured per-request → per-folder →
per-collection via the canonical AuthBlock inheritance chain.

Implementation is deferred to a follow-up iteration; the canonical
AuthBlock type is defined now to reserve the field layout.

## Alternatives Considered

- **Keep AWS SigV4 hard-wired, add other auth as one-off cases**:
  rejected because each new auth type requires executor changes and
  there is no composition story for hook-based refresh.
- **OAuth2 as a built-in authenticator**: deferred, not rejected.
  Hook-based auth already covers the OAuth2 credential-fetch step;
  a built-in would add convenience but is not required for correctness.

## Consequences

**Positive:**
- Any auth scheme expressible as "set headers before request" is
  supported without changing core executor code.
- Hook-based auth reuses the existing hooks infrastructure.
- canonical.AuthBlock field layout is stable from day one.

**Negative / Trade-offs:**
- Token caching policy is unresolved (per-session vs keychain);
  deferred to implementation iteration.
- Hook-based OAuth2 requires users to write a small shell script;
  no GUI credential flow.
