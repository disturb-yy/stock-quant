# Authentication Implementation Control

## Goal

Implement the authentication contract by resolving the local Wayfinder tickets in dependency order, then hand the confirmed code changes to an independent security-aware code review.

## Constraints

- Work one Wayfinder ticket at a time; do not begin a blocked ticket.
- The active ticket is a HITL decision ticket. Do not edit application code until its answer is recorded and accepted.
- Authentication, authorization, public API, and storage changes require an independent `reviewing-code` review after implementation.
- Do not implement OAuth/OIDC, MFA, social login, multi-tenant authorization, or a user-management console in this effort.

## Phase Plan

1. Resolve identity and onboarding decisions.
2. Resolve access and refresh token lifecycle.
3. Resolve RBAC and the API access matrix.
4. Resolve security/error rules and publish the client contract.
5. Route confirmed implementation slices to `coding-project`, validate them, and obtain an independent code review.

## Task Board

| Ticket | Phase | State | Ownership | Dependencies |
|---|---|---|---|---|
| Define User Identity and Onboarding | Context / grilling | Resolved | Codex | None |
| Set Access and Refresh Token Lifecycle | Context / grilling | Resolved | Codex | Define User Identity and Onboarding |
| Specify RBAC and API Access Matrix | Context / grilling | Ready to claim | Unassigned | Define User Identity and Onboarding |
| Set Authentication Security and Error Baseline | Context / grilling | Ready to claim | Unassigned | Define User Identity and Onboarding; Set Access and Refresh Token Lifecycle |
| Publish the Frontend–Backend Authentication Contract | Context / grilling | Blocked | Unassigned | Token lifecycle; RBAC; security/error baseline |
| Decide Credential Recovery and Password Change Scope | Context / grilling | Ready to claim | Unassigned | Define User Identity and Onboarding |

## Decision Trace

- The implementation destination is a frontend–backend contract for local registration/login, JWT Access Tokens, rotating cookie-based Refresh Tokens, `user`/`admin` RBAC, and secure error handling.
- Confirmed: Email is required and unique and is the only sign-in identifier; Username is optional, unique, and only used for display.
- Confirmed: self-registered Users are active immediately with the `user` Role; a Disabled User cannot sign in; passwords permit 12 to 128 characters without composition rules; administrators are provisioned only by controlled operations.
- Confirmed: Access Tokens expire after 15 minutes; an Authenticated Session has a 30-day absolute lifetime and expires after 7 days of inactivity.
- Confirmed: Access Tokens use HS256 with a backend-only `JWT_SIGNING_KEY` of at least 32 random bytes. A `kid` supports a short overlap of the current and previous signing key during rotation.
- Confirmed: Access Tokens contain only `sub`, `sid`, `role`, `iat`, `exp`, `jti`, `iss`, and `aud`; they contain no Email, Username, or other personal information.
- Confirmed: Refresh Tokens are opaque and are stored only as hashes in a persistent Authenticated Session record. Every refresh rotates the token atomically; replay revokes the full session and requires a new login.
- Confirmed: the first release is a same-site frontend/API deployment. The Refresh Token uses a host-only `HttpOnly`, production-`Secure`, `SameSite=Strict` cookie scoped to `/api/v1/auth`, with a 30-day maximum age; only local HTTP development may omit `Secure`.
- Confirmed: `POST /api/v1/auth/logout` revokes and clears only the current session and returns `204` idempotently; there is no first-release all-devices logout endpoint.
- The next eligible decisions are RBAC, security/error baseline, and credential recovery/change scope.

## Review Gates

- No code may be accepted without relevant tests or build validation.
- Any authentication or authorization code change must be reviewed by an independent actor using `reviewing-code`; the implementation actor cannot self-review.

## Event Log

| Date | Actor | Event | Evidence | Next action |
|---|---|---|---|---|
| 2026-08-18 | Codex | Claimed **Define User Identity and Onboarding** | [Ticket](./tickets/auth-01-identity-onboarding.md) | Ask the ticket's first decision frontier. |
| 2026-08-18 | User | Confirmed Email as the required, unique sign-in identifier and Username as an optional display identifier | [Domain language](../../CONTEXT.md) | Decide activation, account states, password policy, and administrator provisioning. |
| 2026-08-18 | User and Codex | Resolved **Define User Identity and Onboarding** | [Resolution](./tickets/auth-01-identity-onboarding.md#resolution) | Select one newly eligible ticket. |
| 2026-08-18 | Codex | Claimed **Set Access and Refresh Token Lifecycle** | [Ticket](./tickets/auth-02-token-session.md) | Decide the token lifetime policy. |
| 2026-08-19 | User | Confirmed Access/Refresh lifetime policy, HS256 signing-key policy, and minimal Access Token claims | [Ticket](./tickets/auth-02-token-session.md) | Decide Refresh Token storage, rotation, and replay handling. |
| 2026-08-19 | User | Confirmed Refresh Token storage, rotation, and replay handling | [Domain language](../../CONTEXT.md) | Decide Refresh Token cookie transport. |
| 2026-08-19 | User | Confirmed same-site Refresh Token cookie policy | [Ticket](./tickets/auth-02-token-session.md) | Decide logout scope and idempotency. |
| 2026-08-19 | User and Codex | Resolved **Set Access and Refresh Token Lifecycle** | [Resolution](./tickets/auth-02-token-session.md#resolution) | Select one newly eligible ticket. |

## Validation Log

- No source code has been modified in this run; only tracker, control, and domain-language artifacts were updated.

## Unresolved Risks

- The RBAC access matrix, security/error baseline, and recovery/change scope remain undecided, so the public API contract is incomplete.
- The repository has no existing authentication implementation or frontend behavior to use as a compatibility reference.
