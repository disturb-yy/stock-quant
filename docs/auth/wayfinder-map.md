---
id: auth-alignment-map
title: Authentication Contract Alignment Map
labels:
  - wayfinder:map
status: open
tracker: local-markdown
---

# Authentication Contract Alignment Map

## Destination

Produce an agreed, implementation-ready frontend–backend authentication contract for Stock Quant. It covers local registration and login, JWT access tokens, refresh sessions, current-user lookup, RBAC, and consistent authentication errors.

## Notes

- Domain: identity, authentication, session, and authorization.
- Resolve every `wayfinder:grilling` ticket with the `grilling` and `domain-modeling` skills; record its answer in that ticket before closing it.
- Agreed baseline: local self-registration with username/email and password; JWT Access Tokens in the `Authorization: Bearer` header; rotating Refresh Tokens in `HttpOnly`, `Secure` cookies; `user` and `admin` Roles; password hashing, login rate limiting, token redaction, and authentication audit logging.
- [Shared authentication language](../../CONTEXT.md) is authoritative for domain terminology.
- The frontend directory currently has no application code, so client behavior is specified here rather than implemented in this effort.

## Decisions so far

<!-- Closed child tickets are indexed here. A ticket holds the full decision and the map only links to it. -->

- [Define User Identity and Onboarding](./tickets/auth-01-identity-onboarding.md) — Email is the required sign-in identity; registered Users are active `user`s with a bounded password policy, and administrators are provisioned only through controlled operations.
- [Set Access and Refresh Token Lifecycle](./tickets/auth-02-token-session.md) — 15-minute minimal-claim JWTs are backed by rotating, hashed, same-site Refresh Token sessions with bounded lifetime and idempotent current-device logout.

## Not yet specified

- Whether multi-device session management, external identity providers, or MFA are needed after the core session lifecycle is specified.

## Out of scope

- Implementing backend code, database migrations, or a frontend UI; this map ends with a contract suitable for those delivery efforts.
- Email verification in the first release; OAuth/OIDC, MFA, social login, an administrative user-management console, and multi-tenant authorization.
