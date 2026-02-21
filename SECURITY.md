# Security Policy

## Supported Versions

The following versions of Ailloy are currently supported with security updates:

| Version | Supported          |
| ------- | ------------------ |
| 0.x.x   | :white_check_mark: |

As Ailloy is currently in alpha stage, all 0.x.x releases receive security updates. Once we reach 1.0, we will establish a more detailed support policy.

## Reporting a Vulnerability

We take security vulnerabilities seriously. If you discover a security issue, please report it responsibly.

### How to Report

**Please do NOT report security vulnerabilities through public GitHub issues.**

Instead, please report them via one of these methods:

1. **GitHub Security Advisories** (Preferred): Use the [Security Advisories](https://github.com/nimble-giant/ailloy/security/advisories/new) feature to privately report the vulnerability.

2. **Email**: Send details to the repository maintainers (contact information available in the repository).

### What to Include

When reporting a vulnerability, please include:

- A description of the vulnerability
- Steps to reproduce the issue
- Potential impact of the vulnerability
- Any suggested fixes (if you have them)

### What to Expect

- **Initial Response**: We aim to acknowledge receipt within 48 hours
- **Status Updates**: We will provide updates on the progress of addressing the vulnerability
- **Resolution Timeline**: We aim to release patches for critical vulnerabilities within 7 days, and for other vulnerabilities within 30 days
- **Disclosure**: We will coordinate with you on public disclosure timing

## Security Update Process

### How Patches Are Released

1. Security fixes are developed in a private branch
2. A new patch version is released with the fix
3. Release notes will credit the reporter (unless they prefer to remain anonymous)

### How Users Are Notified

- Security advisories are published on the [GitHub Security Advisories](https://github.com/nimble-giant/ailloy/security/advisories) page
- Critical updates are noted in release notes
- Users watching the repository will receive notifications

## Security Best Practices for Users

When using Ailloy:

- **Keep Ailloy updated** to the latest version
- **Store sensitive configuration** (API keys, tokens) in environment variables, not in configuration files
- **Review blanks** before using them in your workflow
- **Do not commit** `flux.yaml` files containing sensitive information

## Scope

This security policy applies to:

- The Ailloy CLI tool
- Official blanks distributed with Ailloy
- Documentation and example configurations

Third-party blanks or plugins are not covered by this policy.

## Recognition

We appreciate the security research community's efforts in helping keep Ailloy secure. Reporters of valid security issues will be acknowledged in our release notes (with permission).

---

Thank you for helping keep Ailloy and its users safe!
