---
id: auth-02-token-session
title: Set Access and Refresh Token Lifecycle
parent: ../wayfinder-map.md
labels:
  - wayfinder:grilling
status: closed
assignee: codex
run: 1
resolved_at: 2026-08-19
blocked_by:
  - auth-01-identity-onboarding
---

# Set Access and Refresh Token Lifecycle

## Question

What claims, signing-key ownership, lifetimes, rotation rules, storage record, cookie attributes, logout semantics, and replay handling govern the Access Token and Refresh Token for the agreed User identity?

## Resolution

- Access Tokens expire after 15 minutes. An Authenticated Session expires after 30 days in total or after 7 days without use, whichever occurs first.
- Access Tokens use HS256. The backend alone holds a randomly generated `JWT_SIGNING_KEY` of at least 32 bytes outside version control. Tokens carry `kid`; key rotation briefly accepts only the current and immediately previous key.
- Access Tokens contain only `sub`, `sid`, `role`, `iat`, `exp`, `jti`, `iss`, and `aud`. They do not contain Email, Username, or other personal information.
- A Refresh Token is opaque and high-entropy. The persistent Authenticated Session stores only its hash together with User ID, creation, expiry, last-use, revocation, and replacement information.
- Refresh exchanges rotate the Refresh Token atomically. Replay of an old, revoked, or replaced token revokes the complete Authenticated Session and requires a new login.
- The first release deploys frontend and API on the same site. The Refresh Token is a host-only cookie scoped to `/api/v1/auth` with `HttpOnly`, production-only `Secure`, `SameSite=Strict`, and a 30-day maximum age. Local HTTP development alone may omit `Secure`.
- `POST /api/v1/auth/logout` revokes the current Authenticated Session and clears the cookie. It is idempotent and returns `204` even when the cookie is absent, expired, or already revoked. The first release has no all-devices logout endpoint.

## Evidence

- User confirmed each decision in this conversation on 2026-08-19.
