# Security Policy

## Supported Versions

Aeterna is actively maintained on the `main` branch.

| Version/Branch | Supported |
| --- | --- |
| `main` | Yes |
| Older releases/branches | Best effort |

## Reporting a Vulnerability

Please do **not** open public GitHub issues for security problems.

Use one of these private channels:

1. **GitHub Private Vulnerability Reporting (preferred)**
   - Go to the repository `Security` tab.
   - Click `Report a vulnerability`.
   - Submit details privately to maintainers.

## What to Include

To help us triage quickly, include:

- A clear description of the issue and impact
- Affected component(s) and version/commit
- Reproduction steps or proof-of-concept
- Any logs, stack traces, or screenshots
- Suggested fix or mitigation (if known)

## Response Timeline

We aim to:

- Acknowledge valid reports within **72 hours**
- Share initial triage within **7 days**
- Provide a fix timeline based on severity and complexity

## Disclosure Policy

Please allow us time to investigate and patch before public disclosure.
We will coordinate disclosure timing with the reporter whenever possible.

## Scope

This policy covers vulnerabilities in:

- `backend/` (Go API and services)
- `frontend/` (React web app)
- Deployment files and scripts (`docker-compose*`, `install.sh`, etc.)

Out of scope:

- General support questions or feature requests
- Vulnerabilities only in third-party services you run externally