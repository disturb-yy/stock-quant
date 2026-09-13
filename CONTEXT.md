# Stock Quant

This context defines the shared business language for the Stock Quant service. The authentication terms below apply across the backend domain and its future client contract.

## Authentication

**User**:
A person with a required Email and local Credential who can authenticate to use the service.
_Avoid_: Account, member, ordinary user

**Email**:
The required, unique identifier by which a User signs in.
_Avoid_: Login name, email address field

**Username**:
An optional, unique display identifier for a User that is not used to sign in.
_Avoid_: Login name, account name

**Active User**:
A User whose identity is enabled and may authenticate to use the service.
_Avoid_: Verified user, enabled account

**Disabled User**:
A User whose identity is not permitted to authenticate to use the service.
_Avoid_: Deleted user, inactive account

**Administrator**:
A User assigned the `admin` Role and therefore allowed to perform administrative actions.
_Avoid_: Superuser, root user

**Credential**:
A secret or identifier that proves a User's identity during authentication.
_Avoid_: Login information, account data

**Access Token**:
A short-lived bearer proof representing an authenticated User and their current Role.
_Avoid_: Token, session token

**Refresh Token**:
A long-lived, opaque secret that continues a User's authenticated session by obtaining a new Access Token.
_Avoid_: Persistent login, long token

**Authenticated Session**:
The time-bounded continuation of a User's authentication, represented by a Refresh Token and ending when its absolute or idle lifetime expires.
_Avoid_: Permanent login, token pair

**Revoked Session**:
An Authenticated Session that can no longer issue Access Tokens or be refreshed.
_Avoid_: Logged-out token, deleted session

**Role**:
A named access class assigned to a User; the initial Roles are `user` and `admin`.
_Avoid_: User type, permission level

**Permission**:
A specific action that an authenticated User is allowed to perform.
_Avoid_: Role, access level
