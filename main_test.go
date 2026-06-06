package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestInitCurrentAndOffset(t *testing.T) {
	repo := initRepo(t)

	assertTreehouseOutput(t, repo, "0\n", "init")
	assertTreehouseOutput(t, repo, "0\n", "current")
	assertTreehouseOutput(t, repo, "8080\n", "offset", "8080")
	assertFileContent(t, storePath(t, repo), "0\n")
	assertNoFile(t, lockPathForRepo(t, repo))
}

func TestInitIsIdempotentWithoutForce(t *testing.T) {
	repo := initRepo(t)

	assertTreehouseOutput(t, repo, "4\n", "init", "--set", "4")
	assertTreehouseOutput(t, repo, "4\n", "init", "--set", "8")
	assertTreehouseOutput(t, repo, "8\n", "init", "--set", "8", "--force")
}

func TestInitUsesLowestAvailableNumberAcrossWorktrees(t *testing.T) {
	repo := initRepo(t)
	linked := filepath.Join(t.TempDir(), "linked")
	gitCommand(t, repo, "worktree", "add", linked)

	assertTreehouseOutput(t, repo, "0\n", "init")
	assertTreehouseOutput(t, linked, "1\n", "init")

	if _, err := os.Stat(filepath.Join(linked, ".treehouse")); !os.IsNotExist(err) {
		t.Fatalf("expected no checkout-local .treehouse file, stat err: %v", err)
	}
	assertFileContent(t, storePath(t, linked), "1\n")

	if err := os.Remove(storePath(t, repo)); err != nil {
		t.Fatal(err)
	}
	assertTreehouseOutput(t, repo, "0\n", "init")
}

func TestCurrentRequiresInit(t *testing.T) {
	repo := initRepo(t)

	result := runTreehouse(t, repo, "current")

	if result.err == nil {
		t.Fatal("expected current to fail before init")
	}
	if !strings.Contains(result.stderr, "not initialized") {
		t.Fatalf("expected not initialized error, got stderr:\n%s", result.stderr)
	}
}

func TestInitTimesOutWhenLockIsHeld(t *testing.T) {
	repo := initRepo(t)
	lock := lockPathForRepo(t, repo)

	if err := os.WriteFile(lock, []byte("held by test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	originalWorkingDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(originalWorkingDir); err != nil {
			t.Fatal(err)
		}
	})

	_, err = acquireInitLock(10 * time.Millisecond)

	if err == nil {
		t.Fatal("expected init to fail while lock is held")
	}
	if !strings.Contains(err.Error(), "timed out waiting for init lock") {
		t.Fatalf("expected lock timeout error, got: %v", err)
	}
}

func TestMain(m *testing.M) {
	code := m.Run()
	if testBinaryDir != "" {
		_ = os.RemoveAll(testBinaryDir)
	}
	os.Exit(code)
}

var testBinary string
var testBinaryDir string

type commandResult struct {
	stdout string
	stderr string
	err    error
}

func assertTreehouseOutput(t *testing.T, cwd string, expected string, args ...string) {
	t.Helper()
	result := runTreehouse(t, cwd, args...)
	if result.err != nil {
		t.Fatalf("treehouse %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), result.err, result.stdout, result.stderr)
	}
	if result.stdout != expected {
		t.Fatalf("treehouse %s stdout = %q, want %q", strings.Join(args, " "), result.stdout, expected)
	}
}

func runTreehouse(t *testing.T, cwd string, args ...string) commandResult {
	t.Helper()
	binary := buildTestBinary(t)
	command := exec.Command(binary, args...)
	command.Dir = cwd
	var stdout, stderr strings.Builder
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	return commandResult{stdout: stdout.String(), stderr: stderr.String(), err: err}
}

func buildTestBinary(t *testing.T) string {
	t.Helper()
	if testBinary != "" {
		return testBinary
	}

	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not resolve test source path")
	}
	root := filepath.Dir(sourceFile)
	dir, err := os.MkdirTemp("", "treehouse-test-*")
	if err != nil {
		t.Fatal(err)
	}
	testBinaryDir = dir
	binary := filepath.Join(dir, "treehouse-test")

	command := exec.Command("go", "build", "-o", binary, ".")
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed: %v\n%s", err, output)
	}
	testBinary = binary
	return testBinary
}

func initRepo(t *testing.T) string {
	t.Helper()
	repo := filepath.Join(t.TempDir(), "repo")
	if err := os.Mkdir(repo, 0o755); err != nil {
		t.Fatal(err)
	}

	gitCommand(t, repo, "init")
	gitCommand(t, repo, "config", "user.email", "treehouse@example.com")
	gitCommand(t, repo, "config", "user.name", "treehouse")
	gitCommand(t, repo, "config", "commit.gpgsign", "false")

	readme := filepath.Join(repo, "README.md")
	if err := os.WriteFile(readme, []byte("test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommand(t, repo, "add", "README.md")
	gitCommand(t, repo, "commit", "-m", "initial")

	return repo
}

func storePath(t *testing.T, cwd string) string {
	t.Helper()
	gitDir := gitCommand(t, cwd, "rev-parse", "--path-format=absolute", "--git-dir")
	return filepath.Join(gitDir, ".treehouse")
}

func lockPathForRepo(t *testing.T, cwd string) string {
	t.Helper()
	gitDir := gitCommand(t, cwd, "rev-parse", "--path-format=absolute", "--git-common-dir")
	return filepath.Join(gitDir, ".treehouse.lock")
}

func assertNoFile(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %s not to exist, stat err: %v", path, err)
	}
}

func assertFileContent(t *testing.T, path string, expected string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != expected {
		t.Fatalf("%s = %q, want %q", path, content, expected)
	}
}

func gitCommand(t *testing.T, cwd string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = cwd
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output))
}
