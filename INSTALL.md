# Local installation

The outstanding TUI release tickets do not block installation. `make verify` passed.

## Replace old Codex skills and install all current skills

```bash
cd /Users/eimantas.zilis/repos/agent-toolkit

toolkit_backup="$(mktemp -d "$HOME/agent-toolkit-backup.XXXXXX")"
toolkit_skills="${AGENTS_HOME:-$HOME/.agents}/skills/agent-toolkit"

if [[ -e "$toolkit_skills" || -L "$toolkit_skills" ]]; then
  mv "$toolkit_skills" "$toolkit_backup/agent-toolkit"
fi

printf 'Old skills backed up to: %s\n' "$toolkit_backup"
bash ./scripts/install.sh --provider codex --bundle all
```

This replaces the old skills, including retired `plan-conventions` and `wrap-up`,
with all 29 current skills. Global instructions stay unchanged.

Start a new Codex session. Keep this repository at its current path.

## Install for Claude (optional)

```bash
bash ./scripts/install.sh --provider claude --bundle all --no-global-guidance
```

## Uninstall

```bash
cd /Users/eimantas.zilis/repos/agent-toolkit
make uninstall
```

Removes managed skills and guidance for both providers. Modified files and
retired skills may remain; the backup-and-move step above removes all old Codex
toolkit skills from discovery.
