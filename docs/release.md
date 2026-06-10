# Release And Homebrew

This project can be installed from any directory with `go install` once the repository is pushed to GitHub:

```sh
GOPATH="$HOME/.cache/go" GOBIN="$HOME/.local/bin" go install github.com/baburyx/lazyprd@latest
```

Make sure `GOBIN` is on `PATH`:

```sh
export PATH="$HOME/.local/bin:$PATH"
```

Then run from any project:

```sh
lazyprd
lazyprd --project /path/to/project
```

## Homebrew Tap Flow

Homebrew formulae normally live in a separate tap repository. For this app, create:

```text
github.com/baburyx/homebrew-tap
```

Then copy `packaging/homebrew/lazyprd.rb` to:

```text
homebrew-tap/Formula/lazyprd.rb
```

Release steps:

1. Push this repository to `github.com/baburyx/lazyprd`.
2. Tag a version:

```sh
git tag v0.1.0
git push origin v0.1.0
```

3. Create a GitHub release for `v0.1.0`.
4. Get the source tarball SHA:

```sh
curl -L https://github.com/baburyx/lazyprd/archive/refs/tags/v0.1.0.tar.gz | shasum -a 256
```

5. Update the formula `url`, `sha256`, and `version`.
6. Push the formula to `baburyx/homebrew-tap`.

Users can then install from any directory:

```sh
brew tap baburyx/tap
brew install lazyprd
lazyprd --project /path/to/project
```

## Cross-Platform Notes

- macOS/Linux: Homebrew or `go install`.
- Windows: `go install github.com/baburyx/lazyprd@latest`, then ensure `%USERPROFILE%\go\bin` or your chosen `GOBIN` is on `PATH`.
- Clipboard support depends on platform tools: `pbcopy`, `wl-copy`, `xclip`, `xsel`, or `clip`.

## Before Publishing

- Confirm the GitHub owner/repo in `go.mod`, `README.md`, and `packaging/homebrew/lazyprd.rb`.
- Add a license file if this will be public.
- Run `make verify`.
