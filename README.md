# lazyprd

Keyboard-driven TUI for browsing local `/to-prd` output at `.scratch/*/PRD.md`.

`lazyprd` is task-first: it opens the most recent PRD, selects the first unchecked implementation task, previews the task context, and lets you toggle Markdown checkboxes in-place.

## Run

```sh
go run .
```

Use another project root:

```sh
go run . --project /path/to/project
```

## Install Locally

```sh
GOPATH="$HOME/.cache/go" GOBIN="$HOME/.local/bin" go install .
```

Ensure the install directory is on `PATH`:

```sh
export PATH="$HOME/.local/bin:$PATH"
```

Then run from any directory:

```sh
lazyprd --project /path/to/project
```

From inside a target project, omit `--project`:

```sh
lazyprd
```

## Install From GitHub

```sh
GOPATH="$HOME/.cache/go" GOBIN="$HOME/.local/bin" go install github.com/baburyx/lazyprd@latest
```

## Homebrew

The Homebrew formula template is in `packaging/homebrew/lazyprd.rb`.

Once a GitHub release exists and the formula has the release SHA, users can install with:

```sh
brew tap baburyx/tap
brew install lazyprd
```

See `docs/release.md` for the release and tap setup steps.

## Development

```sh
make test
make build
make install
make verify
```

## Keys

- `Tab` / `Shift+Tab`: cycle panes
- `Ctrl+h/j/k/l`: move focus or selection
- `j` / `k`: move selection
- `/`: fuzzy filter focused list pane
- `Esc`: clear filter or prompt preview
- `Space`: toggle selected implementation task checkbox
- `p`: preview an agent prompt for the selected task
- `y`: copy the prompt, or selected preview content
- `r`: rescan PRDs
- `q`: ask before quitting
