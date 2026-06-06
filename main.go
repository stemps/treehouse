package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const storeFilename = ".treehouse"

type worktree struct {
	path   string
	gitDir string
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "treehouse: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}

	switch args[0] {
	case "init":
		number, err := runInit(args[1:])
		if err != nil {
			return err
		}
		fmt.Println(number)
	case "current":
		if len(args) != 1 {
			return errors.New("usage: treehouse current")
		}
		number, err := readCurrentNumber()
		if err != nil {
			return err
		}
		fmt.Println(number)
	case "offset":
		if len(args) != 2 {
			return errors.New("usage: treehouse offset <base>")
		}
		base, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("base must be an integer: %s", args[1])
		}
		number, err := readCurrentNumber()
		if err != nil {
			return err
		}
		fmt.Println(base + number)
	case "-h", "--help", "help":
		printUsage()
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}

	return nil
}

func runInit(args []string) (int, error) {
	flags := flag.NewFlagSet("init", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)

	force := flags.Bool("force", false, "replace an existing stored worktree number")
	selected := flags.String("set", "", "store an explicit non-negative worktree number")

	if err := flags.Parse(args); err != nil {
		return 0, err
	}
	if flags.NArg() != 0 {
		return 0, errors.New("usage: treehouse init [--force] [--set NUMBER]")
	}

	var selectedNumber *int
	if *selected != "" {
		number, err := parseNonNegativeInt(*selected)
		if err != nil {
			return 0, fmt.Errorf("--set must be a non-negative integer")
		}
		selectedNumber = &number
	}

	return initNumber(*force, selectedNumber)
}

func initNumber(force bool, selected *int) (int, error) {
	store, err := currentStorePath()
	if err != nil {
		return 0, err
	}
	if !force {
		if _, err := os.Stat(store); err == nil {
			return readNumberFile(store)
		} else if !errors.Is(err, os.ErrNotExist) {
			return 0, fmt.Errorf("could not stat %s: %w", store, err)
		}
	}

	number := 0
	if selected != nil {
		number = *selected
	} else {
		next, err := nextAvailableNumber()
		if err != nil {
			return 0, err
		}
		number = next
	}

	if err := os.WriteFile(store, []byte(fmt.Sprintf("%d\n", number)), 0o644); err != nil {
		return 0, fmt.Errorf("could not write %s: %w", store, err)
	}
	return number, nil
}

func readCurrentNumber() (int, error) {
	store, err := currentStorePath()
	if err != nil {
		return 0, err
	}
	if _, err := os.Stat(store); errors.Is(err, os.ErrNotExist) {
		return 0, errors.New("current worktree is not initialized; run `treehouse init`")
	} else if err != nil {
		return 0, fmt.Errorf("could not stat %s: %w", store, err)
	}
	return readNumberFile(store)
}

func nextAvailableNumber() (int, error) {
	worktrees, err := listWorktrees()
	if err != nil {
		return 0, err
	}

	var used []int
	for _, wt := range worktrees {
		number, ok, err := readOptionalNumber(filepath.Join(wt.gitDir, storeFilename))
		if err != nil {
			return 0, err
		}
		if ok {
			used = append(used, number)
		}
	}

	sort.Ints(used)
	next := 0
	for _, number := range used {
		if number == next {
			next++
		} else if number > next {
			break
		}
	}
	return next, nil
}

func currentStorePath() (string, error) {
	gitDir, err := currentGitDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(gitDir, storeFilename), nil
}

func currentGitDir() (string, error) {
	return gitPath("rev-parse", "--path-format=absolute", "--git-dir")
}

func listWorktrees() ([]worktree, error) {
	output, err := git("worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}

	var worktrees []worktree
	for _, line := range strings.Split(output, "\n") {
		path, ok := strings.CutPrefix(line, "worktree ")
		if !ok {
			continue
		}
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("could not resolve worktree path %s: %w", path, err)
		}
		gitDir, err := gitPath("-C", absPath, "rev-parse", "--path-format=absolute", "--git-dir")
		if err != nil {
			return nil, err
		}
		worktrees = append(worktrees, worktree{path: absPath, gitDir: gitDir})
	}

	return worktrees, nil
}

func readOptionalNumber(path string) (int, bool, error) {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return 0, false, nil
	} else if err != nil {
		return 0, false, fmt.Errorf("could not stat %s: %w", path, err)
	}
	number, err := readNumberFile(path)
	if err != nil {
		return 0, false, err
	}
	return number, true, nil
}

func readNumberFile(path string) (int, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("could not read %s: %w", path, err)
	}
	text := strings.TrimSpace(string(content))
	if text == "" {
		return 0, fmt.Errorf("%s is empty", path)
	}
	return parseNonNegativeIntFromFile(path, text)
}

func parseNonNegativeIntFromFile(path string, text string) (int, error) {
	number, err := strconv.Atoi(text)
	if err != nil || number < 0 {
		return 0, fmt.Errorf("%s must contain a non-negative integer", path)
	}
	return number, nil
}

func parseNonNegativeInt(text string) (int, error) {
	number, err := strconv.Atoi(text)
	if err != nil || number < 0 {
		return 0, errors.New("must be a non-negative integer")
	}
	return number, nil
}

func gitPath(args ...string) (string, error) {
	output, err := git(args...)
	if err != nil {
		return "", err
	}
	return filepath.Abs(strings.TrimSpace(output))
}

func git(args ...string) (string, error) {
	command := exec.Command("git", args...)
	output, err := command.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return "", errors.New(message)
	}
	return strings.TrimSpace(string(output)), nil
}

func printUsage() {
	fmt.Print(`Usage:
  treehouse init [--force] [--set NUMBER]
  treehouse current
  treehouse offset <base>

Commands:
  init      Assign this Git worktree a unique number.
  current   Print this worktree's assigned number.
  offset    Print BASE plus this worktree's assigned number.
`)
}
