# ADR-0004 — Email OTP is mailbox verification, not NIST-compliant MFA

**Date:** 2026-11-24  
**Status:** Accepted  
**Deciders:** Product owner, architect  

## Context

Both authentication paths (email/password and Google OIDC) require a 6-digit OTP sent to the user's email before an application session is issued. This provides a mailbox-ownership proof step.

NIST SP 800-63B does not accept email-based out-of-band authentication as a compliant second factor because email is not a secure channel and is susceptible to phishing and account compromise.

## Decision

Email OTP is used as a **mailbox verification and additional friction step**, not as phishing-resistant MFA. The product must never claim NIST MFA compliance for this mechanism.

The implementation follows the OWASP MFA cheat sheet constraints for OTP: 6 digits, 5-minute TTL, maximum 5 incorrect attempts per challenge, resend rate limiting, HMAC-signed digest, single-use, constant-time comparison.

## Consequences

- Future phases may add passkey, TOTP, or hardware-key-based authentication if required.
- Documentation and marketing must not describe this as two-factor or multi-factor authentication without qualification.
- The security baseline in `security.md` remains fully mandatory.
