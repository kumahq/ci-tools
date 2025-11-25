package github

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/term"
)

const (
	ghCommand = "gh"

	envGitHubToken    = "GITHUB_TOKEN"
	envGitHubAPIToken = "GITHUB_API_TOKEN"
	envGHToken        = "GH_TOKEN"
)

var (
	ErrGitHubTokenNotFound = errors.New("GitHub token not found: use --use-gh-auth flag or set GITHUB_TOKEN or GH_TOKEN environment variable")
	ErrGHNotInstalled      = errors.New("'gh' command not found in PATH")
	ErrGHAuthFailed        = errors.New("failed to get token from 'gh auth token'")
	ErrGHAuthEmptyToken    = errors.New("'gh auth token' returned empty output")
)

// isInteractive determines whether user interaction is possible.
// Returns false in CI environments, piped contexts, and non-terminal sessions.
func isInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// isGHAvailable checks whether GitHub CLI is installed and accessible.
func isGHAvailable() bool {
	_, err := exec.LookPath(ghCommand)
	return err == nil
}

// promptYesNo presents a yes/no question to the user with 'Y' as the default.
// Automatically adapts to terminal capabilities: uses single-keystroke input when
// possible, falls back to line-based input otherwise. Only accepts 'y', 'Y', 'n', 'N',
// or empty input (defaults to yes). Re-prompts on invalid input.
func promptYesNo(question string) bool {
	stdinFd := int(os.Stdin.Fd())

	if !term.IsTerminal(stdinFd) {
		return readLineBasedAnswer(question)
	}

	oldState, err := term.MakeRaw(stdinFd)
	if err != nil {
		return readLineBasedAnswer(question)
	}

	defer func() {
		_ = term.Restore(stdinFd, oldState)
	}()

	return readSingleChar(question)
}

// readLineBasedAnswer handles prompts when raw terminal mode is unavailable.
// Empty input defaults to yes. Only accepts y/Y/n/N or empty input. Re-prompts on
// invalid input. Used in non-interactive contexts and as a fallback.
func readLineBasedAnswer(question string) bool {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		_, _ = fmt.Fprintf(os.Stderr, "%s [Y/n]: ", question)

		if !scanner.Scan() {
			return true
		}

		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))

		if answer == "" {
			return true
		}

		if answer == "y" || answer == "yes" {
			return true
		}

		if answer == "n" || answer == "no" {
			return false
		}

		_, _ = fmt.Fprintln(os.Stderr, "Please answer 'y' or 'n'")
	}
}

// readSingleChar enables immediate response without Enter key in terminals.
// Requires terminal to be in raw mode. Only accepts y/Y/n/N or Enter (defaults to yes).
// Re-prompts on invalid input. Uses carriage return to prevent cursor positioning issues.
func readSingleChar(question string) bool {
	var buf [1]byte
	firstPrompt := true

	for {
		if firstPrompt {
			_, _ = fmt.Fprintf(os.Stderr, "%s [Y/n]: ", question)
			firstPrompt = false
		}

		_, err := os.Stdin.Read(buf[:])
		if err != nil {
			_, _ = fmt.Fprint(os.Stderr, "\n")
			return true
		}

		char := buf[0]

		switch char {
		case 'y', 'Y', '\r', '\n':
			_, _ = fmt.Fprintf(os.Stderr, "%c\r\n", char)
			return true
		case 'n', 'N':
			_, _ = fmt.Fprintf(os.Stderr, "%c\r\n", char)
			return false
		default:
			_, _ = fmt.Fprintf(os.Stderr, "\r\033[K")
			_, _ = fmt.Fprintf(os.Stderr, "Invalid input '%c'. Please answer 'y' or 'n': ", char)
			continue
		}
	}
}

// getGitHubToken retrieves authentication using a priority cascade with automatic fallback.
// Tries multiple sources in order: gh CLI (if requested), environment variables, then
// interactive prompt. Returns ErrGitHubTokenNotFound only when all methods are exhausted.
func getGitHubToken(useGHAuth bool) (string, error) {
	// Priority 1: If --use-gh-auth flag is set, try GitHub CLI first
	if useGHAuth {
		if token, _ := tryGHAuth(); token != "" {
			return token, nil
		}
	}

	// Priority 2, 3, 4: Check environment variables
	if token := getTokenFromEnv(); token != "" {
		return token, nil
	}

	// Priority 5: In interactive sessions, offer to use gh auth if available
	if token, err := tryInteractiveGHAuth(useGHAuth); token != "" || err != nil {
		return token, err
	}

	return "", ErrGitHubTokenNotFound
}

// tryGHAuth attempts GitHub CLI authentication with graceful failure.
func tryGHAuth() (string, error) {
	token, err := getTokenFromGHAuth()
	if err == nil {
		return token, nil
	}

	_, _ = fmt.Fprintf(os.Stderr, "Warning: failed to get token from gh auth, falling back to environment variables: %v\n", err)

	return "", nil
}

// getTokenFromEnv checks standard GitHub token environment variables.
// Prioritizes GITHUB_TOKEN over GITHUB_API_TOKEN over GH_TOKEN for consistency.
func getTokenFromEnv() string {
	if token := os.Getenv(envGitHubToken); token != "" {
		return token
	}

	if token := os.Getenv(envGitHubAPIToken); token != "" {
		return token
	}

	if token := os.Getenv(envGHToken); token != "" {
		return token
	}

	return ""
}

// tryInteractiveGHAuth offers GitHub CLI auth as a last resort in terminals.
// Only activates when not already tried via flag, in interactive sessions, with gh available.
func tryInteractiveGHAuth(useGHAuth bool) (string, error) {
	if useGHAuth {
		return "", nil
	}

	if !isInteractive() || !isGHAvailable() {
		return "", nil
	}

	if !promptYesNo("No GitHub token found. Use 'gh auth token' to authenticate?") {
		return "", nil
	}

	token, err := getTokenFromGHAuth()
	if err == nil {
		return token, nil
	}

	_, _ = fmt.Fprintf(os.Stderr, "Warning: failed to get token from gh auth: %v\n", err)

	return "", nil
}

// getTokenFromGHAuth executes GitHub CLI to retrieve an authenticated token.
// Requires gh CLI to be installed and user to be authenticated via 'gh auth login'.
func getTokenFromGHAuth() (string, error) {
	if !isGHAvailable() {
		return "", ErrGHNotInstalled
	}

	token, err := executeGHAuthToken()
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrGHAuthFailed, err)
	}

	if token == "" {
		return "", ErrGHAuthEmptyToken
	}

	return token, nil
}

// executeGHAuthToken runs the gh CLI command to retrieve the token.
func executeGHAuthToken() (string, error) {
	cmd := exec.Command(ghCommand, "auth", "token")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}
