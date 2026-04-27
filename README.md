# juju-helpers

A small CLI utility that makes day-to-day [Juju](https://juju.is) development faster.

## Commands

### `seed`

Injects a set of Juju shell aliases into `~/.bashrc` and `~/.zshrc`. Running it multiple times is safe — the managed block is replaced, never duplicated.

**Aliases added:**

| Alias | Expands to |
|-------|-----------|
| `js` | `juju status` |
| `jss` | `juju status --watch 2s --relations` |
| `jm` | `juju models` |
| `jsc` | `juju switch` |
| `nuke` | `juju destroy-model --force --no-prompt --destroy-storage --no-wait` |
| `jh` | `juju-helpers` |

```bash
juju-helpers seed
# seeded /home/user/.bashrc
# seeded /home/user/.zshrc
```

Reload your shell afterwards:

```bash
source ~/.bashrc   # or source ~/.zshrc
```

---

### `nuke-all`

Destroys all Juju models whose names begin with `test-` or `jubilant-`. The currently active model is skipped by default.

```bash
# Preview and confirm interactively
juju-helpers nuke-all

# Skip confirmation prompt
juju-helpers nuke-all -y

# Also destroy the currently active model
juju-helpers nuke-all --include-current

# Show help
juju-helpers nuke-all --help
```

**Example output:**

```
Models to destroy:
  test-foo
  jubilant-5b7a99c7
  jubilant-d5583396
(skipping current model "jubilant-7e88abdc" — use --include-current to include it)

Destroy 3 model(s)? [y/N] y
nuking test-foo ... ok
nuking jubilant-5b7a99c7 ... ok
nuking jubilant-d5583396 ... ok
```

## Installation

Requires Go 1.18+.

```bash
go install github.com/alithethird/juju-helpers@latest
```

The binary is placed in `$GOPATH/bin` (or `~/go/bin` by default). Make sure that directory is on your `PATH`.
