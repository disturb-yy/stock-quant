---
id: auth-01-identity-onboarding
title: Define User Identity and Onboarding
parent: ../wayfinder-map.md
labels:
  - wayfinder:grilling
status: closed
assignee: codex
run: 1
resolved_at: 2026-08-18
blocked_by: []
---

# Define User Identity and Onboarding

## Question

What exact identity fields, registration validation, password policy, and initial Role assignment define a local User? Decide whether username, email, or both are required and unique; whether an activation step is required; and which account states prevent login.

## Run control

See the [authentication implementation control record](../implementation-control.md) for the current phase, dependencies, review gate, and validation state.

## Resolution

- Email is required, unique, and is the sole sign-in identifier; Username is optional, unique, and is only a display identifier.
- A self-registered User is immediately an Active User. Email verification is not part of the first release.
- An Active User and a Disabled User are the initial identity states; a Disabled User cannot sign in. Session-revocation behavior is deferred to the token and security decisions.
- A password must contain 12 to 128 characters. The first release does not require character-class composition rules; password hashing is decided by the security ticket.
- Self-registration always assigns the `user` Role. The `admin` Role may only be granted by deployment-time initialization or a controlled operations process; there is no public role-escalation endpoint.

## Evidence

- User confirmed the complete decision frontier in this conversation on 2026-08-18.
