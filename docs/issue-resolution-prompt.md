# Issue Resolution Prompt

Use this prompt when fixing upstream open-source issues end-to-end.

---

## The Prompt

You are fixing upstream GitHub issues from `<owner>/<repo>`. For each issue:

### 1. Understand Before Touching

Read the issue body, linked code, and any referenced PRs. Before writing a single line of code, answer:
- What is the exact failure mode? (wrong behavior, panic, missing feature)
- Which files own the behavior? (grep/glob — don't guess)
- Are there existing tests that should cover this but don't?

### 2. Branch Isolation

Create a dedicated git worktree for each issue:
```bash
git worktree add /tmp/<repo>-<issue-id> -b <type>/ss-<issue-id> main
```
Work exclusively in that directory. Never `git checkout` in the user's working tree.

### 3. Minimal, Targeted Fix

Write the smallest change that fixes the root cause. No opportunistic refactors. No "while I'm here" changes. If you notice something unrelated, open a separate issue.

### 4. Multi-Pass Review (run all four in sequence, not just the first one that passes)

**Pass 1 — Correctness**: Does the logic handle all inputs correctly? Off-by-ones, nil checks, missing returns, wrong variable used.

**Pass 2 — Security**: Is mutable state stored where immutable state is needed? (e.g. username → user ID to avoid TOCTOU races). Does any auth path allow bypass? Are permissions validated at the right layer?

**Pass 3 — Observability**: Does a successful operation log at the right level? Does a failed attempt log a warning? Are errors wrapped with context? Check both the happy path AND the error paths — missing success logs are as problematic as missing error logs.

**Pass 4 — Concurrency / Edge cases**: Can this race? Is there a TOCTOU window between a lookup and an action? What happens with empty input, duplicate input, or input that was valid at check time but invalid at use time?

### 5. Write Tests First (or alongside)

Use the existing test framework (testscript/txtar for integration, `_test.go` for unit). Cover:
- The exact scenario described in the issue
- The negative case (wrong key, expired token, missing permission)
- Edge cases the issue implies but doesn't state (e.g. key with spaces, token for deleted user)

Run the full test suite — not just the new tests — before declaring done.

### 6. Commit with Context

```
<type>(<scope>): <what changed>

<why: root cause in one sentence>
<how: the fix in one sentence>

Closes #<issue-id>
```

### 7. PR Against the Right Target

```bash
gh pr create \
  --base dev \
  --title "<type>: <short description> (#<issue-id>)" \
  --body "$(cat <<'EOF'
## Problem
<one paragraph: what was broken and why>

## Fix
<one paragraph: what changed and the key design decision>

## Tests
- [ ] <test case 1>
- [ ] <test case 2>

Closes charmbracelet/soft-serve#<issue-id>
EOF
)"
```

### 8. Final Validation Checklist

Before marking a PR ready:

- [ ] `go build ./...` passes
- [ ] `go test ./...` passes (all, not just new tests)
- [ ] No new compiler warnings or lint errors
- [ ] Success path logs at DEBUG/INFO
- [ ] Failure path logs at WARN with enough context to debug
- [ ] Auth decisions happen at one layer only (not duplicated across middleware + handler)
- [ ] No mutable state used as an identity token (use IDs, not usernames)
- [ ] PR description explains *why*, not just *what*

---

## Anti-Patterns We Avoided

| Anti-pattern | What we did instead |
|---|---|
| Storing username in SSH permissions extension | Stored immutable user ID; resolved on use |
| `AccessLevelByPublicKey` called per-repo in a loop | Single `AccessLevelForUser` on the context user |
| Merging all trailing args unconditionally | Only merge when `-k` flag is present; error otherwise |
| Copy shortcut on every TUI tab (conflicted with pane handlers) | Scoped shortcut to the Readme tab only |
| `initializePermissions` mutating a local copy | Added `ctx.SetValue` to persist back to context |
| Reviewing only the changed function | Traced the entire call chain from auth → middleware → handler |
