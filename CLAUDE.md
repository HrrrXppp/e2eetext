# Working conventions for this repo

## Sub-agents

- Prefer delegating multi-step, research-heavy, or independently-verifiable work to sub-agents (the `Agent` tool) rather than doing it all inline, especially for: broad codebase exploration, parallel investigation across unrelated files/areas, and isolated implementation steps that can be checked independently.
- Keep concurrent sub-agents under 20 at any one time. Fan out in batches rather than launching everything at once.
- Sub-agents that edit files run in their own git worktree (isolated directory), not the main checkout, so parallel agents can't step on each other's changes. This is enforced by `.claude/settings.json`'s `worktree.bgIsolation: "worktree"`.

## GitHub issue/PR ticket processing (recurring workflow)

- The main cycle thread never reads or interprets ticket *content* — no fetching a specific issue/PR's comment bodies, no deciding whether something is an approval vs. feedback vs. a question, no drafting plan text, no implementing, no posting comments. All of that is "working the ticket" and must happen inside a per-ticket `Agent` call, every time, with no exceptions for "just this once it's quick." Each such agent edits files/runs git, so it automatically gets its own isolated git worktree per the setting above.
- The main cycle thread never touches local git branches either — no `git fetch`/`checkout`/`reset`, not even to keep local `dev` in sync. Reconnaissance is pure GitHub API (`curl`) reading issue/PR metadata; it needs no local checkout at all. Each dispatched agent already fetches `origin/dev` itself before branching (in its own isolated worktree), so a main-thread dev sync is redundant work done in the wrong place, not a safety measure.
- The main cycle thread's only jobs: list open issue/PR numbers and cheap metadata (title, comment *count*, review *count* — numbers only, never comment bodies) to detect which tickets changed since last cycle; dispatch one `Agent` per changed/new ticket to read it, decide what it means, and act; report the dispatch list. Reading an issue's title to route it to an agent is fine; reading its comment text to decide what to do with it is not.
- Batch independent tickets in parallel (multiple issues needing fresh plans, or multiple PRs needing responses, in the same cycle) rather than processing them one at a time in sequence — still capped under 20 concurrent per the sub-agent rule above.
- Give each agent full self-contained context in its prompt (issue/PR number, the GitHub token, what output format to return) since it starts with no memory of the cycle's other tickets, and it — not the main thread — is the one that fetches and reads the actual comment content.
- Every GitHub comment posted by an agent — a new plan, a plan revision, a PR review reply, a plain answer, a PR description — must be clearly marked/signed "Claude". This applies to anything that shows up as text on GitHub's issue/PR pages. It does NOT apply to git commit messages: those stay unsigned (no "Co-Authored-By" or any other attribution trailer), per `.claude/settings.json`'s `attribution.commit: ""`.
- A push is not done until CI is green. Passing local checks (`go build`/`go vet`/`go test`, `npm test`, `npm run lint`, `terraform fmt`/`validate`, etc.) is necessary but not sufficient — an agent that pushes a commit must also check the resulting GitHub Actions run status for that PR/branch (e.g. `GET /repos/{owner}/{repo}/commits/{sha}/check-runs` or `GET .../actions/runs`) and treat a red or still-pending CI run as part of the same task, not a separate future ticket. If CI (including e2e) fails after a push, the same agent should investigate the actual CI logs and fix it before finishing, rather than reporting success on local checks alone.

## Documentation

- Keep `README.md` current: whenever a change adds/removes/alters a documented command, workflow, file layout, environment variable, or API endpoint, update the corresponding `README.md` section in the same change. Don't leave it describing stale behavior.

## Code quality

- We shouldn't add a method which is used only in tests to production code. If something is needed purely for test setup/assertions (a seam, a getter, a constructor variant), put it in the test file, a `_test.go`/`.test.ts` helper, or test-only fixture code — not in the production source file. Before adding a new exported method to production code, check whether any non-test caller actually needs it.
