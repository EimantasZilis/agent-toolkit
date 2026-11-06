# Local installation

Install the selected skills from this checkout. Setup saves your choices for
later refreshes; keep the checkout for linked resources.

```bash
make install
```

For Bash setup without Go, use `make install-cli`. To select all skills for both
providers without prompts:

```bash
bash ./scripts/install.sh --provider codex --provider claude --bundle all
```

Add `--full-template true` to install the global guidance template, or
`--concise true` to enable standing concise-output instructions. Start a new
agent session after setup.

## Refresh or remove

```bash
make update
make uninstall
```

Update refreshes the saved selection. Uninstall removes verified owned skills,
guidance, and saved state. Ownership conflicts stop either operation before
writes. See [setup and recovery instructions](README.md#installation-instructions)
for details.
