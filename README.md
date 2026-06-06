# Treehouse

`treehouse` helps you isolate your development environments when using Git
worktrees.

It assigns a stable number for each worktree, so you can use this number to
derive per-worktree local configuration like ports, database names, etc... or
anything you want isolated per worktree.

For example, the command

```bash
PORT="$(treehouse offset 3000)" npm run dev
```

will start your dev server on port 3000 on the first worktree, port 3001 on the
second, etc...

The worktree number is stored in `.treehouse` inside the worktree's local Git
metadata directory, as reported by `git rev-parse --git-dir`. For linked
worktrees this is usually under the repository's common `.git/worktrees/`
directory, so it is not checked in and is removed with the worktree metadata.

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
$ treehouse init
0  # sets the initial worktree number
$ treehouse current
0  # outputs the current worktree's number
$ treehouse offset 8080
8080  # increments the given number by the worktree's number
$ ./treehouse run sh -c 'echo "This is worktree $WORKTREE_NUMBER"'
This is worktree 0  # runs a command with WORKTREE_NUMBER set
```

In another worktree:

```bash
$ treehouse init
1
$ treehouse current
1
$ treehouse offset 8080
8081
$ ./treehouse run sh -c 'echo "This is worktree $WORKTREE_NUMBER"'
This is worktree 1
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

Node app with worktree-aware configuration:

```bash
treehouse run npm run dev
```

## All Commands

- `treehouse init`: assign the lowest unused non-negative worktree number.
- `treehouse init --set 7`: explicitly store worktree number `7`.
- `treehouse init --force`: replace an existing stored number.
- `treehouse current`: print the current worktree number.
- `treehouse offset <base>`: print `<base> + current worktree number`.
- `treehouse run <command>`: run `<command>` with `WORKTREE_NUMBER` set to
  the current worktree number.

`current`, `offset`, and `run` fail if the worktree has not been initialized.
Use `treehouse init` first.

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
