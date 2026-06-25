# Contributing to Leave Management System

Thanks for your interest in contributing. This document covers the workflow, conventions, and checklist expected for any pull request, across both the backend and frontend repositories.

## Table of Contents

- [Before You Start](#before-you-start)
- [Development Workflow](#development-workflow)
- [Branch Naming](#branch-naming)
- [Commit Convention](#commit-convention)
- [Pull Request Checklist](#pull-request-checklist)
- [Pull Request Template](#pull-request-template)
- [Code Style](#code-style)
- [Project Structure Quick Reference](#project-structure-quick-reference)
- [Writing Tests](#writing-tests)
- [Reporting Bugs](#reporting-bugs)
- [Suggesting Features](#suggesting-features)
- [Reporting Security Issues](#reporting-security-issues)
- [Code Review Etiquette](#code-review-etiquette)
- [Getting Help](#getting-help)

## Before You Start

For anything beyond a small fix — new features, breaking changes, or anything that changes existing behavior — please open an issue first to discuss the approach. This avoids wasted effort on a pull request that doesn't fit the project's direction.

**Good first issues** are labeled `good first issue` on GitHub — these are scoped, low-risk changes that are a good way to get familiar with the codebase.

## Development Workflow

### 1. Fork the repository

Click **Fork** on GitHub to create your own copy of the repository.

### 2. Clone your fork

```bash
git clone https://github.com/<your-username>/<repository-name>.git
cd <repository-name>
```

Add the original repository as an upstream remote, so you can keep your fork in sync:

```bash
git remote add upstream https://github.com/Zenithive/LeaveManagementSystem.git
```

### 3. Create a branch

Always branch off the latest `main`:

```bash
git fetch upstream
git checkout main
git merge upstream/main
git checkout -b feature/leave-balance-export
```

See [Branch Naming](#branch-naming) below for naming conventions.

### 4. Implement your changes

- Keep changes focused — one logical change per pull request. A PR that fixes a bug *and* refactors an unrelated function is harder to review and harder to revert if something goes wrong.
- Follow the existing code style and project structure (see [Code Style](#code-style))
- Add or update tests for any behavior you change
- Update documentation if your change affects setup, configuration, or usage

**Example — a focused change:**

Say you're adding a new leave type ("Sabbatical") to the policy engine. A well-scoped PR would touch:
```
internal/models/leavePolicy.go         # add the new leave type constant
internal/service/leave/leavePolicy.go  # add validation rules for it
internal/repositories/leavePolicy.go   # persist/query the new type
pkg/notification/templates/leave.html  # mention the new type in emails, if relevant
internal/handler/leavePolicy_test.go   # test the new type is accepted
```
It would *not* also rename unrelated variables in `leaveApprovalFlow.go` just because you were in the file — that goes in a separate `refactor/` PR.

### 5. Run tests locally

**Backend (Go):**

```bash
go test ./...
go vet ./...
gofmt -l .          # lists any files that aren't formatted correctly
```

If `gofmt -l .` prints file names, run `gofmt -w .` to fix formatting before committing.

**Frontend (React/TypeScript):**

```bash
npm run lint
npm run test
npm run build        # confirms the production build still succeeds
```

Fix any failures before opening a pull request — CI runs the same checks and blocks merges on failure.

### 6. Commit your changes

Follow the [Commit Convention](#commit-convention) below. Keep commits small and focused — it's easier to review five clear commits than one large one, and easier to `git revert` a single commit if something breaks.

```bash
git add internal/service/leave/leavePolicy.go internal/models/leavePolicy.go
git commit -m "feat: add sabbatical leave type to policy engine"
```

### 7. Push and create a pull request

```bash
git push origin feature/leave-balance-export
```

Open a pull request against `main` on the upstream repository. Fill in the [PR template](#pull-request-template), including what changed and why. Link any related issue (e.g. `Closes #42`).

### 8. Keep your branch up to date

If `main` has moved on while your PR is open, rebase rather than merge, to keep history clean:

```bash
git fetch upstream
git rebase upstream/main
git push --force-with-lease origin feature/leave-balance-export
```

### 9. Address review feedback

A maintainer will review your PR and may request changes. Push additional commits to the same branch — they'll appear in the same PR automatically. Once approved, a maintainer will merge it — please don't merge your own PR unless explicitly asked to.

## Branch Naming

Use the following prefixes, followed by a short, hyphenated description:

| Prefix | Use for | Example |
|---|---|---|
| `feature/` | New functionality | `feature/leave-balance-export` |
| `fix/` | Bug fixes | `fix/leave-approval-race-condition` |
| `docs/` | Documentation only | `docs/update-installation-guide` |
| `refactor/` | Code changes with no behavior change | `refactor/optimize-leave-query` |
| `test/` | Adding or improving tests only | `test/add-notification-dispatcher-tests` |
| `chore/` | Tooling, dependencies, CI config | `chore/bump-go-version` |

Branch names should be lowercase, hyphen-separated, and descriptive enough to understand the change without opening the PR.

**Good:** `fix/leave-withdrawal-double-email`
**Avoid:** `fix/bug`, `patch-1`, `my-changes`

## Commit Convention

This project follows [Conventional Commits](https://www.conventionalcommits.org/). Each commit message should be structured as:

```
<type>: <short description>
```

| Type | Use for |
|---|---|
| `feat` | A new feature |
| `fix` | A bug fix |
| `docs` | Documentation changes only |
| `refactor` | Code change that neither fixes a bug nor adds a feature |
| `test` | Adding or correcting tests |
| `chore` | Maintenance tasks (dependency bumps, tooling, config) |
| `style` | Formatting changes with no code logic change |
| `perf` | A change that improves performance |

**Examples drawn from this project's actual domains:**

```
feat: add leave balance calculation
feat: add Slack birthday announcement scheduler
fix: correct leave approval issue
fix: prevent duplicate email on leave withdrawal
fix: resolve dial tcp [::1]:5432 connection error in Docker
docs: update installation guide
docs: add docker-compose configuration for local Postgres
refactor: optimize query handling
refactor: split notification service into email and slack providers
test: add approval workflow tests
test: add EventProcessor routing coverage for unknown event types
chore: bump golang-jwt to v4.5.2
```

**Commit body (optional, for non-trivial changes):**

```
fix: prevent duplicate email on leave withdrawal

OnLeaveWithdrawn was calling both renderAndSendRecipients and
renderAndSendBulk for the same HR recipient list, causing each HR
user to receive two emails. Removed the redundant renderAndSendBulk
call and consolidated to a single send path.

Closes #58
```

Keep the description line in the imperative mood ("add", not "added" or "adds"), under ~72 characters. Add detail in the body after a blank line when the "why" isn't obvious from the diff alone.

## Pull Request Checklist

Before requesting review, confirm:

- [ ] **Tests passing** — `go test ./...` (backend) or `npm run test` (frontend) succeeds locally
- [ ] **Lint passing** — `go vet ./...` + `gofmt -l .` (backend) or `npm run lint` (frontend) reports no issues
- [ ] **No secrets added** — no API keys, passwords, tokens, or `.env` files committed; run `git diff` and check carefully before pushing
- [ ] **Documentation updated** — README, `.env.example`, or relevant docs updated if your change affects setup, configuration, or usage
- [ ] **Feature tested** — manually verified the change works as intended, not just that automated tests pass
- [ ] **No unrelated changes** — the diff only contains what the PR description says it contains

Pull requests that don't meet this checklist may be asked to update before review continues.

## Pull Request Template

When opening a PR, structure the description like this:

```markdown
## What does this PR do?
Brief description of the change.

## Why is this needed?
Link to the issue, or explain the motivation if there isn't one.

## How was this tested?
- [ ] Unit tests added/updated
- [ ] Manually tested locally
- [ ] Tested with Docker Compose

## Screenshots (if UI change)
Before/after, if applicable.

## Checklist
- [ ] Tests passing
- [ ] Lint passing
- [ ] No secrets added
- [ ] Documentation updated
```

## Code Style

**Backend (Go):**
- Run `gofmt` (or rely on your editor's format-on-save) before committing
- Follow the existing layering convention: `routes` → `middleware` → `handler` → `service` → `repositories`
- Business logic belongs in `service/`, not in `handler/` or `repositories/` — handlers should stay thin (parse request → call service → return response)
- Keep `pkg/` reserved for code that's reusable independent of the core domain (security, notifications, constants); domain-specific logic stays under `internal/`
- Error handling: wrap errors with context (`fmt.Errorf("leave: approve request: %w", err)`) rather than returning bare errors
- Use the existing `slog` logger for structured logging — avoid introducing a second logging library

**Frontend (React/TypeScript):**
- Follow the existing ESLint configuration — `npm run lint` should pass cleanly with zero warnings
- Match the existing component structure under `src/components/`, `src/pages/`, `src/services/`, etc.
- Prefer the existing UI primitives (shadcn/ui) over introducing new component libraries
- Use `react-hook-form` + `zod` for any new forms, matching the existing pattern
- API calls belong in `src/services/`, not inline inside components

## Project Structure Quick Reference

**Backend**
```
routes → middleware → handler → service → repositories
```
A new endpoint typically touches: `routes/` (register the route), `handler/` (parse/validate request), `service/` (business logic), `repositories/` (DB access), and `internal/models/` (if new fields are needed).

**Frontend**
```
pages → components → services → backend API
```
A new screen typically touches: `src/pages/` (the route's page component), `src/components/` (any new reusable pieces), and `src/services/` (the API call it depends on).

## Writing Tests

- **Backend:** test files live next to the file they test, same package, suffix `_test.go` (e.g. `leavePolicy.go` → `leavePolicy_test.go`). Mock external services (`EmailProvider`, Slack webhooks) — never call Resend or Slack live in a test. See `pkg/notification/testsupport` for an example mock provider.
- **Frontend:** colocate component tests with the component (e.g. `Button.tsx` → `Button.test.tsx`). Mock API calls in `src/services/` rather than hitting a real backend.

If you're adding a feature, add at least one test that would fail without your change. If you're fixing a bug, add a regression test that reproduces the bug first, then fix it.

## Reporting Bugs

If you find a bug that isn't a security issue (see below), open a GitHub issue with:
- A clear, descriptive title
- Steps to reproduce
- Expected vs. actual behavior
- Environment details (OS, Go/Node version, Docker or local, browser if frontend)
- Relevant logs or error output, if any

## Suggesting Features

Open an issue describing the problem you're trying to solve and your proposed approach. This lets maintainers and other contributors weigh in before any code is written. Check [ROADMAP.md](./ROADMAP.md) first — your idea may already be planned for a future version.

## Reporting Security Issues

**Do not** report security vulnerabilities through public GitHub issues. See [SECURITY.md](./SECURITY.md) for the responsible disclosure process.

## Code Review Etiquette

**As a contributor:**
- Don't take review feedback personally — it's about the code, not you
- If you disagree with a suggestion, explain your reasoning rather than just reverting it silently
- Respond to review comments even if just to say "done" or "good point, fixed"

**As a reviewer (if you're reviewing others' PRs):**
- Be specific — "this could cause a nil pointer if `d.Recipients` is empty" is more useful than "this looks risky"
- Distinguish blocking issues from suggestions (e.g. prefix optional nits with "nit:")
- Approve once the PR meets the checklist, even if it's not how you'd have written it yourself

## Getting Help

If you're stuck setting up the project or unsure how something works, open a [Discussion](../../discussions) (if enabled) or an issue tagged `question`, rather than guessing — it also helps future contributors who hit the same problem.

---

Thank you for contributing — every fix, feature, and documentation improvement helps the project.
