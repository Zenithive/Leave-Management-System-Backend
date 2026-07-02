# Security Policy

## Supported Versions

The following versions of this project currently receive security updates:

| Version | Supported |
|---|---|
| 1.x (latest `main`) | ✅ |
| < 1.0 (pre-release) | ❌ |

This table will be updated as new major versions are released.

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues, discussions, or pull requests.** Publicly disclosing a vulnerability before a fix is available puts every user of this project at risk.

If you discover a security vulnerability in this project — in either the backend (Go) or frontend (React) — please report it privately by emailing:

**security@zenithive.com**

When reporting, please include as much of the following as you can:

- A clear description of the vulnerability and its potential impact
- Steps to reproduce it (proof-of-concept code or requests are welcome)
- The affected component (e.g. a specific endpoint, handler, or frontend route)
- The version or commit hash you tested against
- Any suggested mitigation, if you have one

You should receive an automated confirmation that your email was received. A maintainer will follow up personally as outlined in the SLA below.

If your GitHub repository has [private vulnerability reporting](https://docs.github.com/en/code-security/security-advisories/guidance-on-reporting-and-writing/privately-reporting-a-security-vulnerability) enabled, you may use that instead of email — look for the **Report a vulnerability** button under the **Security** tab of the repository.

## Response SLA

| Stage | Target |
|---|---|
| **Initial acknowledgement** | Within 72 hours of report |
| **Triage & severity assessment** | Within 7 days of report |
| **Target remediation** | Based on severity (see below) |
| **Public disclosure** | Coordinated with the reporter, after a fix is released |

### Remediation targets by severity

We use [CVSS v3.1](https://www.first.org/cvss/calculator/3.1) (or equivalent judgment for issues that don't map cleanly to CVSS) to assess severity and prioritize remediation:

| Severity | Target remediation time |
|---|---|
| **Critical** (e.g. remote code execution, auth bypass, mass data exposure) | Within 7 days |
| **High** (e.g. privilege escalation, significant data leakage) | Within 14 days |
| **Medium** (e.g. limited-scope data exposure, CSRF on a non-critical action) | Within 30 days |
| **Low** (e.g. minor information disclosure, best-practice hardening) | Next scheduled release |

These are targets, not guarantees — complex issues may take longer, and we'll keep you informed of progress throughout.

## Disclosure Policy

We follow a **coordinated disclosure** process:

1. You report the vulnerability privately via the email above.
2. We acknowledge receipt and begin triage.
3. We work on a fix, keeping you updated on progress.
4. Once a fix is released, we coordinate with you on public disclosure timing.
5. With your permission, we credit you in the release notes or a security advisory, unless you prefer to remain anonymous.

We ask that you do not publicly disclose the vulnerability until a fix has been released and we've agreed on a disclosure date together.

## Scope

This policy covers vulnerabilities in:

- The backend API (authentication, authorization, leave/approval logic, database access, notification delivery)
- The frontend application (XSS, CSRF, insecure data handling, dependency vulnerabilities)
- The Docker images and Docker Compose configuration shipped in this repository
- CI/CD workflows defined in `.github/workflows/`

The following are generally **out of scope** unless they demonstrate a concrete, exploitable impact:

- Vulnerabilities requiring physical access to a user's device
- Issues in outdated, unsupported versions (see [Supported Versions](#supported-versions))
- Vulnerabilities in third-party services this project integrates with but does not control (e.g. Resend, Slack, the hosting platform) — please report those directly to the respective vendor
- Denial-of-service reports based purely on request volume rather than an application-level flaw
- Missing security headers or best-practice suggestions with no demonstrated exploit (these are welcome as a regular GitHub issue, not a security report)

## Security Best Practices for Self-Hosters

If you're deploying this project yourself:

- Never commit `.env` files or real credentials — use `.env.example` as a template only
- Generate `CRON_SECRET` and any JWT signing secrets as random bytes (e.g. `openssl rand -hex 32`), never as a memorable string
- Use `sslmode=require` for production database connections, not `sslmode=disable`
- Keep dependencies up to date — this repository uses Dependabot to flag known vulnerabilities; review and merge security update PRs promptly
- Restrict `ALLOWED_ORIGINS` (CORS) to your actual frontend domain(s) in production — avoid wildcards

## Acknowledgements

We're grateful to everyone who responsibly discloses vulnerabilities. With permission, security researchers who report valid issues will be credited here and in the relevant release notes.

_No vulnerabilities have been publicly disclosed yet._
