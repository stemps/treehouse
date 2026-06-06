# Treehouse

`treehouse` assigns a stable number to each Git worktree, so you can use this
number to derive per-worktree local configuration like ports, database names,
etc... anything you want isolated per worktree.

The worktree number is stored in `.treehouse` inside the worktree's local Git
metadata directory, as reported by `git rev-parse --git-dir`. For linked
worktrees this is usually under the repository's common `.git/worktrees/`
directory, so it is not checked in and is removed with the worktree metadata.

`treehouse init` uses a repo-wide `.treehouse.lock` file in the Git common
directory while it scans worktrees and writes the assigned number. By default it
waits up to 10 seconds for the lock.

## Installation

With Homebrew:

```bash
brew install stemps/tap/treehouse
```

With Go:

```bash
go install github.com/stemps/treehouse@latest
```

## Usage

```bash
treehouse init
treehouse current
treehouse offset 8080
```

Example:

```bash
$ treehouse init
0  # sets the initial worktree number
$ treehouse current
0  # outputs the current worktree's number
$ treehouse offset 8080
8080  # increments the given number by the worktree's number
```

In another worktree:

```bash
$ treehouse init
1
$ treehouse current
1
$ treehouse offset 8080
8081
```

### Typical Setups

Rails/Puma port:

```bash
bin/rails server -p "$(treehouse offset 3000)"
```

Rails database name suffix in `config/database.yml`:

```yaml
development:
  adapter: postgresql
  database: my_app_development_<%= `treehouse current`.strip %>
```

Node app port:

```bash
PORT="$(treehouse offset 3000)" npm run dev
```

Docker published port:

```bash
docker run --rm -p "$(treehouse offset 8080):80" nginx
```

## All Commands

- `treehouse init`: assign the lowest unused non-negative worktree number.
- `treehouse init --set 7`: explicitly store worktree number `7`.
- `treehouse init --force`: replace an existing stored number.
- `treehouse current`: print the current worktree number.
- `treehouse offset <base>`: print `<base> + current worktree number`.

`current` and `offset` fail if the worktree has not been initialized. Use
`treehouse init` first.

## Development

```bash
go test ./...
go build -o treehouse .
```

## Release

Releases are created from `main` with:

```bash
brew install semver
make release
```

The release task asks for a version number, writes it to `VERSION`, commits that
change, creates an annotated tag like `v0.1.0`, and pushes the commit and tag.
The tag triggers GitHub Actions to publish release artifacts and update
`stemps/homebrew-tap`.

`VERSION` starts at `0.0.0`; enter `0.1.0` when cutting the first public
release. After that, `make release` defaults to the next patch version. Release
versions may use full SemVer syntax, such as `0.2.0-alpha.1`, but should be
entered without the leading `v`.

Go install users can request prereleases explicitly, for example
`go install github.com/stemps/treehouse@v0.2.0-alpha.1`. `@latest` generally
prefers stable releases over prereleases.

The `stemps/treehouse` repository needs a `HOMEBREW_TAP_GITHUB_TOKEN` secret
with contents write access to `stemps/homebrew-tap`.
