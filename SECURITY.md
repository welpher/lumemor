# Security Policy

## Reporting a vulnerability

**Please do not open public issues for security problems.**

- Use GitHub's private vulnerability reporting on this repository, or
- Contact a maintainer directly

We aim to respond within 7 days. We appreciate responsible disclosure and will credit reporters who wish to be named.

## Scope

lumemor is pre-release: there are no supported production versions yet. Once releases exist, supported versions will be listed here.

## What we care about especially

- Anything that could expose photos or metadata across users (visibility/ACL bypasses)
- Authentication and session handling
- Share-link token predictability
- The remote-access path (pairing, certificate pinning, relay)
