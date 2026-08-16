# Security Review: new-api

## Scope

Standard repository-wide security audit of the NookMux LLM gateway at /workspace/new-api. Reviewed Go backend routes, middleware auth, relay/proxy, payment webhooks, OAuth/passkey/2FA flows, stored media, and web frontend DOM sinks.

- Scan mode: deep_repository
- Target kind: git_worktree
- Target ID: target_sha256_0152d45ef1f01a40cb2ce0e67e04ee1608bc3b6f9b8b26baf8d01fdd1f7a5f72
- Revision: da0f1241a211c245890d82bec00ff33826ccd259
- Snapshot digest: codex-security-snapshot/v1:sha256:779c49cb769398b4d6dfef8042e74a39b56262768c4e6fe859ec3dbc09720225
- Inventory strategy: repository
- Included paths: .
- Excluded paths: none
- Runtime or test status: not recorded
- Artifacts reviewed: AGENTS.md, Dockerfile, README.md, VERSION, common/, constant/, controller/, dto/, go.mod, go.sum, i18n/, logger/, main.go, makefile, middleware/, model/, nookmux.service, oauth/, pkg/, relay/, router/, scripts/, service/, setting/, types/
- Scan context: Standard single-pass security audit of the NookMux Go repository (LLM API gateway/proxy with multi-provider relay, billing, OAuth, passkey/2FA auth, admin console, stored media). Worker label discovery-0001; subagents=3. No inherited SECURITY.md policy; no user context; no knowledge base.

Limitations and exclusions:
- Relay channel adaptor sink review (packet C/D) was partially delegated to an investigator that became unavailable; remaining review performed sequentially by the coordinator, covering api_request.go, stored asset handlers, custom_voice upload, and web DOM sinks. Node web frontend was reviewed at a source-signal level (dangerouslySetInnerHTML/redirect/open) but not every component file was individually audited. Vendored Go module source was consulted for the CORS library only.
- Excluded web/node_modules/\*\*: vendored dependencies, not product source
- Excluded web/dist/\*\*: build output, embedded
- Excluded logs/\*\*: runtime log files
- Excluded .serena/\*\*: IDE cache
- Excluded .git/\*\*: version control metadata
- Excluded docs/\*\*: documentation, reviewed as untrusted analysis input only

### Scan Summary

| Field | Value |
| --- | --- |
| Reportable DSS findings | 2 |
| Report instances | 2 |
| Report severity mix | medium: 1, low: 1 |
| Report confidence mix | high: 2 |
| Coverage | partial |
| Validation mode | static_source_only |

Canonical artifacts: `scan-manifest.json`, `findings.json`, and `coverage.json`. This report is a deterministic projection of those files.

## Threat Model

NookMux is a Go (Gin) multi-provider LLM API gateway and billing platform. It proxies OpenAI/Claude/Gemini/other model requests using admin-configured upstream channels, enforces token-based and session-based authentication with role tiers (guest/common/admin/root), manages user quota and top-ups (Epay/Stripe/redemption), supports OAuth login (GitHub/LinuxDO) and passkey/2FA, stores user-uploaded media (images/videos) served via HMAC-signed URLs, and provides an admin console (React SPA). Trust boundaries: unauthenticated network attacker ↔ public API/relay/MCP-asset routes; authenticated common-user token/session ↔ user-scoped API routes; admin ↔ admin-scoped routes; root ↔ system option/db-migration routes; payment platform ↔ anonymous webhook endpoints; upstream LLM provider ↔ relay response parsing.

### Assets

- User and admin session cookies (30-day signed cookies)
- API access tokens (Authorization header bearer / sk- tokens)
- User quota/billing ledger (topups, redemptions, transfers)
- Channel provider API keys and credentials
- Stored media (images/videos) owned by users
- System configuration options (OptionMap, RootAuth-gated)
- OAuth provider bindings and identity linkage

### Trust Boundaries

- Unauthenticated network ↔ public API routes (GET /api/status, /api/setup, /api/user/login, /api/oauth/\*, webhooks, /mcp/image|video signed URLs)
- API token / session cookie ↔ UserAuth-guarded routes
- Admin session/token ↔ AdminAuth-guarded routes (channels, users, logs, tokens, stored-media admin)
- Root session/token ↔ RootAuth-guarded routes (options, db migration, dynamic ratio rules)
- Payment platform (Epay/Stripe) ↔ anonymous webhook callbacks (POST /api/stripe/webhook, /api/user/epay/notify)
- Upstream LLM provider ↔ relay response parsers and streaming handlers
- User-controlled notification URLs (webhook/bark/gotify) ↔ server-side outbound HTTP requests

### Attacker Capabilities

- Unauthenticated network actor reaching public API and relay routes
- Authenticated common user with valid API token or session
- Authenticated admin (non-root) with admin-level session/token
- Payment platform webhook sender (Epay/Stripe) sending signed or forged callbacks
- Malicious or compromised upstream LLM provider returning crafted responses
- Cross-origin browser actor interacting with the web frontend

### Security Objectives

- Prevent authentication bypass, session fixation, and OAuth state CSRF
- Enforce role hierarchy: common \< admin \< root for privileged mutations
- Prevent IDOR and cross-tenant access to tokens, logs, tickets, stored media, topups
- Protect user quota/billing ledger from double-credit, replay, or paid-but-uncredited states
- Prevent SSRF, open redirect, path traversal, and injection in relay/proxy and file handling
- Avoid sensitive credential or configuration exposure to unauthenticated or lower-privileged users

### Assumptions

- No inherited SECURITY.md policy exists; audit applied general secure-design principles.
- The 30-day session cookie lifetime is intentional; Secure is opt-in via COOKIE_SECURE.
- Setup (POST /api/setup) is anonymous by design for first-run initialization; protecting an uninitialized instance is a deployment responsibility.
- Channel base URLs and API keys are admin-configured trusted inputs; user relay requests select among them but do not supply arbitrary upstream hosts.
- SSRF protection (ValidateURL + dial-time recheck) is configurable via FetchSetting; relay upstream calls are admin-configured destinations, not user-supplied URLs.

## Findings

| Findings | Reports | Severity | Confidence | Detailed write-up |
| --- | --- | --- | --- | --- |
| AdminResetPasskey skips role-hierarchy check, allowing a non-root admin to delete a root user's passkey credentials | [occ_f4343b16a101826a81994861](#finding-1) | medium | high | occ_f4343b16a101826a81994861: inline below |
| 2FA login pending session has no time bound, allowing extended-window 2FA brute-force via the 30-day session cookie | [occ_d6438fd51f4bfb01701c7339](#finding-2) | low | high | occ_d6438fd51f4bfb01701c7339: inline below |

### Confidence Scale

| Label | Meaning |
| --- | --- |
| high | Direct evidence supports the finding with no material unresolved blocker. |
| medium | Evidence supports a plausible issue, but material runtime or reachability proof remains. |
| low | Evidence is incomplete and the item is retained only for explicit follow-up. |

<a id="finding-1"></a>

### [1] AdminResetPasskey skips role-hierarchy check, allowing a non-root admin to delete a root user's passkey credentials

| Field | Value |
| --- | --- |
| Severity | medium |
| Confidence | high |
| Confidence rationale | Direct source review confirms AdminResetPasskey (controller/passkey.go:475-509) lacks any role comparison, contrasting with AdminDisable2FA (controller/twofa.go:451) and ManageUser (controller/user.go:762) which enforce `myRole <= targetUser.Role && myRole != common.RoleRootUser`. |
| Category | access_control |
| CWE | CWE-862 |
| Affected lines | controller/passkey.go:475, router/api_router.go:119, controller/twofa.go:451, controller/user.go:762 |

#### Summary

DELETE /api/user/:id/reset_passkey is mounted under AdminAuth (router/api_router.go:119), but AdminResetPasskey (controller/passkey.go:475-509) never compares the caller's role against the target user's role. Every other admin credential-mutation handler enforces `myRole <= targetUser.Role && myRole != common.RoleRootUser` — AdminDisable2FA (controller/twofa.go:451) and ManageUser (controller/user.go:762). AdminResetPasskey loads the target user, counts passkeys, and unconditionally calls model.DeletePasskeyByUserID(user.Id) with no role comparison. A non-root admin can therefore delete a root user's passkey credentials, removing the root's strong authentication factor.

#### Root Cause

AdminResetPasskey was implemented without the role-hierarchy guard that every other admin credential-mutation handler (AdminDisable2FA, ManageUser) enforces: `myRole <= targetUser.Role && myRole != common.RoleRootUser`.

#### Validation

Confirmed by reading controller/passkey.go:475-509: AdminResetPasskey loads the target user, counts passkeys, and calls model.DeletePasskeyByUserID(user.Id) with no role comparison. Contrasted with controller/twofa.go:451 (AdminDisable2FA) and controller/user.go:762 (ManageUser), both of which enforce myRole \<= targetUser.Role && myRole != common.RoleRootUser before proceeding.

#### Dataflow

Authenticated admin (non-root) sends DELETE /api/user/:id/reset_passkey → AdminAuth middleware passes (router/api_router.go:119) → AdminResetPasskey loads target user by :id → counts passkeys → model.DeletePasskeyByUserID(user.Id) called unconditionally with no role check → root user's passkey credentials deleted.

#### Reachability

Any authenticated admin (non-root) can target a root user's :id to delete their passkey credentials. The AdminAuth middleware only checks that the caller has admin-or-higher role; it does not compare caller vs target role.

#### Severity

**Medium** — Privilege escalation: a non-root admin can delete a root user's passkey credentials, bypassing the role hierarchy. Requires admin-level access, limiting the attacker population, but enables compromise of root accounts' strong authentication factors.

Additional runtime or deployment evidence could raise or lower this severity.

#### Remediation

Add a role-hierarchy check in AdminResetPasskey (controller/passkey.go:475-509) matching the pattern used by AdminDisable2FA (controller/twofa.go:451) and ManageUser (controller/user.go:762): reject the request if myRole \<= targetUser.Role or if the target user is a root user (myRole != common.RoleRootUser).

Preventive controls:
- Stripe webhook.ConstructEvent at topup_stripe.go:159
- Atomic WHERE id=? AND status=? CAS in CompleteEpayTopUp and Recharge prevents double-crediting on replay

<a id="finding-2"></a>

### [2] 2FA login pending session has no time bound, allowing extended-window 2FA brute-force via the 30-day session cookie

| Field | Value |
| --- | --- |
| Severity | low |
| Confidence | high |
| Confidence rationale | Source directly shows no timestamp is stored or checked; the contrast with SecureVerification's 300s timeout is clear. |
| Category | authentication_failures |
| CWE | CWE-613 |
| Affected lines | controller/user.go:87, controller/user.go:88, controller/twofa.go:353, main.go:148, controller/secure_verification.go:19 |

#### Summary

Login (controller/user.go:84-93) sets session pending_username/pending_user_id when 2FA is enabled. Verify2FALogin (controller/twofa.go:344-415) only checks that pending_user_id exists; it never records or validates a timestamp. The session cookie has MaxAge 2592000 (30 days, main.go:148). The 2FA lockout resets after each 300s window, so an attacker with a valid first-stage credential can repeatedly attempt TOTP/backup codes over an extended period. The secure-verification step-up path already enforces SecureVerificationTimeout=300s; this login flow lacks the equivalent bound.

#### Root Cause

The 2FA login pending session was implemented without storing a timestamp or enforcing a lifetime, unlike the SecureVerification step-up flow which already enforces a 300-second timeout.

#### Validation

Confirmed by reading controller/user.go:84-93: session.Set("pending_username", user.Username); session.Set("pending_user_id", user.Id) with no timestamp stored. And controller/twofa.go:351-362: pendingUserId := session.Get("pending_user_id"); if pendingUserId == nil {...}; userId, ok := pendingUserId.(int) with no expiry check. main.go:148 sets MaxAge 2592000. controller/secure_verification.go:19 defines SecureVerificationTimeout=300 and enforces it for step-up verification, showing the established pattern the login flow omits.

#### Dataflow

POST /api/user/login → user.ValidateAndFill succeeds → if IsTwoFAEnabled: session.Set("pending_user_id", user.Id) at user.go:88 with no timestamp. POST /api/user/login/2fa → Verify2FALogin reads session.Get("pending_user_id") at twofa.go:353, checks only nil/type, validates TOTP/backup code, calls setupLogin.

#### Reachability

An unauthenticated attacker who knows a user's password can initiate login, receive a 2FA challenge, and then have up to 30 days to submit codes. The 2FA lockout (5 attempts / 300s) resets after each lockout, allowing resumed guessing.

#### Severity

**Low** — Extended brute-force window but 2FA lockout and rate limiting provide partial mitigation; attacker must already know the password. Low likelihood of successful exploitation.

Additional runtime or deployment evidence could raise or lower this severity.

#### Remediation

When setting the pending session, also store a timestamp (e.g. session.Set("pending_user_id_set_at", now)). In Verify2FALogin, reject if now - pending_user_id_set_at \>= 300 (reuse SecureVerificationTimeout). Clear pending fields on timeout.

Tests:
- Unit test: Verify2FALogin with a pending session older than 300 seconds should reject

Preventive controls:
- CriticalRateLimit middleware on POST /api/user/login and /api/user/login/2fa (router/api_router.go:56-57)
- 2FA lockout enforced per-account (model/twofa.go: MaxFailAttempts=5, LockoutDuration=300s)
- TOTP/backup code validation is correct (model/twofa.go)

## Reviewed Surfaces

| Surface | Risk Area | Outcome | Notes |
| --- | --- | --- | --- |
| Authentication & authorization middleware | not recorded | No issue found | No additional canonical notes were recorded. |
| User account & OAuth flows | not recorded | No issue found | No additional canonical notes were recorded. |
| Passkey & 2FA lifecycle | not recorded | Reported | No additional canonical notes were recorded. |
| Token management routes | not recorded | No issue found | No additional canonical notes were recorded. |
| Payment webhooks (Epay/Stripe) & billing ledger | not recorded | Reported | No additional canonical notes were recorded. |
| Ticket system ownership checks | not recorded | No issue found | No additional canonical notes were recorded. |
| Stored media access control & signed URLs | not recorded | No issue found | No additional canonical notes were recorded. |
| System options & admin configuration | not recorded | No issue found | No additional canonical notes were recorded. |
| Relay proxy header passthrough & request construction | not recorded | No issue found | No additional canonical notes were recorded. |
| SSRF protection & outbound HTTP clients | not recorded | No issue found | No additional canonical notes were recorded. |
| Custom voice upload & upstream calls | not recorded | No issue found | No additional canonical notes were recorded. |
| Web frontend DOM sinks (dangerouslySetInnerHTML, redirects) | not recorded | No issue found | No additional canonical notes were recorded. |
| Relay channel adaptors (30+ provider implementations) | not recorded | Needs follow-up | No additional canonical notes were recorded. |
| Web React component tree (990 files) | not recorded | Needs follow-up | No additional canonical notes were recorded. |

## Open Questions And Follow Up

- Second focused investigator unavailable; coordinator performed sequential fallback on highest-risk relay files only
  - Follow-up prompt: Review deferred unit deferred-1ed885dd224b6291 and close its stated proof gap.
- Reviewed at source-signal level (grep for dangerouslySetInnerHTML/redirect/open) but not every component individually audited
  - Follow-up prompt: Review deferred unit deferred-747a60b572d97ccd and close its stated proof gap.
