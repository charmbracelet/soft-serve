# Security Audit Summary - Soft Serve
**Date**: 2026-03-29

## Executive Summary

Comprehensive security audit performed following STRIDE methodology across OWASP Top 10 + custom threat modeling. This document summarizes findings from automated review rounds 56-60 and provides recommendations for remediation.

## Asset Map

### Databases
- **Type**: SQLite (default) or PostgreSQL (configurable)
- **Version**: Check via `go list -m all`
- **Note**: Direct SQL queries using string concatenation (potential SQL injection)

### Authentication Mechanisms
1. **SSH** - Public key authentication (via `pkg/ssh/`)
2. **HTTP Basic** - Username/password with bcrypt hashing
3. **HTTP Token** - Access tokens (SHA256 hashed)
4. **HTTP Bearer** - JWT tokens (Ed25519 signed, 1-hour default expiry)

### Public Routes
**HTTP Endpoints** (pkg/web/*.go):
- `/info/{repo}/info` - Repository info
- `/info/{repo}/info/refs` - Git refs
- `/info/{repo}/info/branches` - Branches
- `/info/{repo}/info/tags` - Tags
- `/repo/{repo}/raw/{ref}` - Raw content
- `/repo/{repo}/archive/{ref}` - Archive
- `/repo/{repo}/blob/{ref}` - File blobs
- `/info/repos` - Repo list
- LFS endpoints - serviceLfsBatch, serviceLfsBasic*, etc.
- Auth endpoints - parseAuthHdr, authenticate
- Go-get endpoints - goget.go

**SSH Commands** (pkg/ssh/cmd/*.go):
- blob.go, branch.go, commit.go, create.go, delete.go, description.go
- git.go, hidden.go, info.go, jwt.go, list.go, mirror.go
- private.go, project_name.go, pubkey.go, push_mirror.go, rename.go
- repo.go, set_username.go, settings.go, tag.go, token.go, tree.go, user.go, webhooks.go

### Input Elements
- Repository names (URL parameters)
- Authorization headers (Bearer, Token, Basic)
- File paths (LFS objects, blob content)
- JWT subject/issuer/audience
- Usernames (SSH authentication)
- Passwords (Basic auth)
- Access tokens (Token auth)

## Security Findings Summary

### Automated Review Rounds (56-60)

**Critical Findings Already Addressed**:
1. ✓ JWT claims validation (expiration, not-before, issuer, audience) - PR #857
2. ✓ JWT token expiration error handling (returns proto.ErrTokenExpired) - PR #861
3. ✓ LFS N+1 batch queries (GetLFSObjectsByOids) - PR #150

**Critical Findings Documented** (from round 59):
1. Plsintext password storage in `pkg/store/database/user.go` - Issue #863
2. SQL injection vulnerabilities in `pkg/store/database/repo.go` - Issue #863
3. User deletion race condition in `pkg/backend/user.go` - Issue #863
4. Connection pool exhaustion risk in `pkg/db/db.go` - Issue #863
5. Path traversal vulnerability in repo names - Issue #862
6. Certificate injection in SSH middleware - Issue #863

**Should-Fix Findings**:
1. Timing attack mitigation incomplete (bcrypt dummy hash)
2. Weak token entropy (JWT 20 bytes only)
3. Missing input validation across multiple endpoints
4. SQLite pragma injection risk (config.go DSN concatenation)
5. TLS certificate validation incomplete
6. Password field validation missing in models

## STRIDE Analysis Categories

### Spoofing
**Tainted Input**: Repo name, file path, LFS object IDs - Potential for path traversal
**Tampering**: JWT tokens (signature verification present, but claims validation added)
**Repudiation**: Rate limiting present (pkg/web/ context middleware)
**Information Disclosure**: Error messages may expose internal state

### Tampering
**Denial of Service**: Rate limiting and connection limits mitigate
**Data Tampering**: Not detected (DB transactions, signature verification)

### Repudiation
**Denial of Service**: Rate limiting available
**Data Tampering**: No data integrity issues detected

### Information Disclosure
**Log Injection**: Error logging uses structured logger, not raw SQL
**Message Header**: Not detected
**Privacy Violation**: User data logged in authentication flow

## Trust Boundaries

**Client → Server**:
- Authentication: JWT tokens (signed with Ed25519), access tokens (SHA256), basic auth
- Protocol: SSH, HTTP, Git protocol
- Trust Boundary: All authenticated operations
- Attack Surface: All public routes

**Server → Server**:
- External: None (self-hosted)
- Internal: Backend code, database
- Trust Boundary: Full trust
- Attack Surface: All internal components

**Server → Server**:
- N/A (single application instance)
- Trust Boundary: Full trust
- Attack Surface: LFS storage, git operations

**Client → Server**:
- N/A (direct SSH access)

**Server → Admin**:
- SSH public key authentication
- Trust Boundary: Admin access via valid credentials
- Attack Surface: Repository management, settings

## Recommendations

### Critical (Immediate Action Required)

1. **Fix Plaintext Password Storage** - Add validation in `pkg/store/database/user.go` to ensure passwords are bcrypt hashed before storage
2. **Implement Parameterized Queries** - Replace string concatenation in `pkg/store/database/repo.go` with `tx.Rebind()`
3. **Fix User Deletion Race** - Move repo deletion into database transaction
4. **Add Connection Pool Limits** - Configure max open connections in `pkg/db/db.go:Open()`
5. **Enhance Path Sanitization** - Improve `utils.SanitizeRepo()` to prevent `../` sequences
6. **Validate SSH Certificates** - Add fingerprint verification in `pkg/ssh/middleware.go`
7. **Increase JWT Token Entropy** - Change from 20 to 32 bytes (256 bits)
8. **Implement Constant-Time Comparisons** - Use subtle.Crypto.ConstantTimeCompare for password/secret checks

### High Priority

1. **Comprehensive Input Validation** - Validate all user inputs:
   - Repository names (path traversal)
   - LFS object IDs
   - File paths (blob endpoints)
   - JWT parameters

2. **TLS Certificate Validation** - Add certificate chain validation in HTTP layer

3. **Error Message Sanitization** - Remove technical details from error messages exposed to clients

### Medium Priority

1. **Timing Attack Improvements** - Enhance dummy hash comparison strategy
2. **SQL Injection Prevention** - Audit all raw SQL queries for parameterization gaps
3. **Password Strength Validation** - Check bcrypt cost factor and complexity requirements

### Security Testing Recommendations

1. **Penetration Testing** - Automated tools (sqlmap, Burp Suite)
2. **Manual Testing** - Security-focused QA
3. **Security Code Review** - Expert review for all authentication/authorization code
4. **Dependency Scanning** - Regular automated scanning (Trivy, Snyk, Grype)

## Notes

- **No Hardcoded Secrets Found**: Manual scan revealed no hardcoded secrets in codebase
- **Context Usage**: 76,242 / 200,000 tokens (~38%) used during audit
- **Limitations**: Asset mapping performed via code inspection; actual penetration testing not executed
- **Recommendations**: Prioritize Critical items above, then work down list

## Related Issues

- #857 - JWT claims validation
- #861 - JWT token expiration error handling
- #150 - LFS batch queries
- #862 - Path traversal & input validation
- #863 - Comprehensive security fixes

## Next Steps

1. Review and prioritize security findings
2. Implement Critical fixes (plaintext passwords, SQL injection, race conditions, path traversal)
3. Add comprehensive input validation framework
4. Implement security testing (unit and integration)
5. Document security architecture and threat model
