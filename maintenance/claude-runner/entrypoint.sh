#!/usr/bin/env bash
# Containerizes the "main cycle thread" role CLAUDE.md defines: fetch the
# repo, run `claude -p` against the ticket-processing prompt, sleep, repeat.
#
# Any arguments passed to `docker run <image> ...` are exec'd directly
# instead of starting the loop, so ad hoc commands work for smoke-testing
# the image (e.g. `docker run --rm <image> claude --version`). The loop
# below only runs when the container is started with no command, which is
# how docker-compose.yml runs it.
set -euo pipefail

if [ "$#" -gt 0 ]; then
	exec "$@"
fi

: "${GITHUB_TOKEN:?GITHUB_TOKEN is required}"
: "${CLAUDE_CODE_OAUTH_TOKEN:?CLAUDE_CODE_OAUTH_TOKEN (or ANTHROPIC_API_KEY) is required}"
: "${GIT_AUTHOR_EMAIL:?GIT_AUTHOR_EMAIL is required}"
: "${REPO_URL:=https://github.com/HrrrXppp/e2eetext.git}"
: "${CYCLE_INTERVAL_SECONDS:=300}"
# dontAsk relies on the committed .claude/settings.json allowlist rather than
# a blanket bypass, so the container never has to guess about a command that
# isn't in that list. `bypassPermissions`/`--dangerously-skip-permissions` is
# an explicit opt-in escape hatch (CLAUDE_PERMISSION_MODE=bypassPermissions)
# for when the allowlist proves too narrow for a new ticket shape -- it is
# never the default.
: "${CLAUDE_PERMISSION_MODE:=dontAsk}"

git config --global user.name "${GIT_AUTHOR_NAME:-Claude (automated)}"
git config --global user.email "$GIT_AUTHOR_EMAIL"
git config --global credential.helper '!f() { echo "username=x-access-token"; echo "password=$GITHUB_TOKEN"; }; f'

[ -d /workspace/e2eetext/.git ] || git clone "$REPO_URL" /workspace/e2eetext
cd /workspace/e2eetext

# Claude Code gates a project's committed permissions.allow entries behind a
# one-time interactive "trust this workspace" dialog, recorded per-path in
# ~/.claude.json. `claude -p ... --permission-mode dontAsk` runs fully
# non-interactively, so that dialog is never shown -- pre-accept it here for
# this exact clone path so the allowlist in .claude/settings.json actually
# takes effect. Runs once per container start; cheap and idempotent, and
# independent of the claude-runner-state volume so it doesn't matter whether
# ~/.claude.json already exists or what else it holds.
node -e '
	const fs = require("fs");
	const path = process.env.HOME + "/.claude.json";
	const config = fs.existsSync(path) ? JSON.parse(fs.readFileSync(path, "utf8")) : {};
	config.projects ||= {};
	config.projects["/workspace/e2eetext"] ||= {};
	config.projects["/workspace/e2eetext"].hasTrustDialogAccepted = true;
	fs.writeFileSync(path, JSON.stringify(config, null, 2));
'

while true; do
	echo "[$(date -Is)] starting cycle"
	git fetch origin
	git checkout dev
	git pull --ff-only
	claude -p "Run the GitHub issue/PR ticket-processing cycle described in CLAUDE.md." \
		--permission-mode "$CLAUDE_PERMISSION_MODE" \
		--output-format text \
		|| echo "[$(date -Is)] cycle failed, will retry next interval" >&2
	echo "[$(date -Is)] cycle done, sleeping ${CYCLE_INTERVAL_SECONDS}s"
	sleep "$CYCLE_INTERVAL_SECONDS"
done
