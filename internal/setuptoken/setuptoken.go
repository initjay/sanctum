// Package setuptoken wraps `claude setup-token`, which mints a long lived
// OAuth token for a Claude subscription account, so sanctum can isolate a
// subscription based profile with an env var instead of the shared macOS
// Keychain login.
package setuptoken

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"unicode"
)

// Result is the outcome of running claude setup-token.
type Result struct {
	// Token is the token parsed out of the command's output, if one was
	// confidently found. Empty if parsing didn't find one.
	Token string
	// RawOutput is everything the command printed to stdout, so a caller
	// can show it to the user and fall back to asking them to paste the
	// token by hand when Token is empty.
	RawOutput string
}

// Run runs `claude setup-token` as a child process scoped to configDir,
// with stdin passed through and stdout/stderr mirrored to passthroughOut
// so the interactive browser based login flow works normally for whoever
// is running it, while also capturing stdout to look for the resulting
// token afterward.
//
// The child's environment is built from scratch rather than inherited
// wholesale: it keeps only the handful of vars a normal shell session
// needs, sets CLAUDE_CONFIG_DIR to configDir, and deliberately excludes
// any ANTHROPIC_API_KEY / CLAUDE_CODE_OAUTH_TOKEN / ANTHROPIC_AUTH_TOKEN
// sanctum's own process might have inherited from an already activated
// profile, so an ambient credential can't short circuit the login prompt
// this is meant to produce a fresh one from.
func Run(ctx context.Context, configDir string, stdin io.Reader, passthroughOut io.Writer) (Result, error) {
	c := exec.CommandContext(ctx, "claude", "setup-token")
	c.Env = childEnv(configDir)
	c.Stdin = stdin
	c.Stderr = passthroughOut

	var captured bytes.Buffer
	c.Stdout = io.MultiWriter(passthroughOut, &captured)

	if err := c.Run(); err != nil {
		return Result{RawOutput: captured.String()}, fmt.Errorf("running claude setup-token: %w", err)
	}

	raw := captured.String()
	return Result{
		Token:     parseToken(raw),
		RawOutput: raw,
	}, nil
}

// childEnv builds the environment for a claude setup-token child process:
// a small allowlist of vars a normal shell session needs, plus
// CLAUDE_CONFIG_DIR pointed at configDir. Everything else from sanctum's
// own process, including any credential vars an already activated profile
// set, is left out on purpose.
func childEnv(configDir string) []string {
	keep := map[string]bool{
		"PATH": true, "HOME": true, "TERM": true, "USER": true,
		"LANG": true, "LC_ALL": true, "SHELL": true, "TMPDIR": true,
	}

	env := make([]string, 0, len(keep)+1)
	for _, kv := range os.Environ() {
		name, _, ok := strings.Cut(kv, "=")
		if ok && keep[name] {
			env = append(env, kv)
		}
	}

	return append(env, "CLAUDE_CONFIG_DIR="+configDir)
}

// parseToken makes a best effort attempt to find the generated token in
// claude setup-token's output. This hasn't been verified against a real
// run, since producing one requires completing an interactive browser
// login. It looks for a line shaped like "<label containing the word
// token>: <value>", on the theory that CLI tools generally print a
// generated credential in a labeled line like that. Because this is
// unverified, callers must never trust the result outright: a non-empty
// return is a candidate for a human to confirm, and an empty one means
// fall back to asking the user to paste the token by hand.
func parseToken(output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)

		// Split on the first colon, not the last: using the label (the
		// part before it) to decide whether this line is even about a
		// token at all is what lets a line like "Login token URL:
		// https://example.com/oauth/authorize?scope=..." get rejected
		// below for looking like a URL, instead of returning everything
		// after the URL's own "https:" as if it were the token.
		idx := strings.Index(line, ":")
		if idx == -1 || idx == len(line)-1 {
			continue
		}

		label := strings.ToLower(line[:idx])
		if !containsWord(label, "token") {
			continue
		}

		value := strings.TrimSpace(line[idx+1:])
		if value == "" || strings.ContainsAny(value, " \t") || len(value) < 20 {
			continue
		}
		if strings.Contains(value, "://") || strings.Contains(value, "/") || strings.HasPrefix(strings.ToLower(value), "http") {
			continue
		}

		return value
	}

	return ""
}

// containsWord reports whether word appears in s as a standalone word
// rather than as a substring of a larger word, so a label like
// "tokenizer" doesn't count but "login token url" does.
func containsWord(s, word string) bool {
	for _, field := range strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		if field == word {
			return true
		}
	}

	return false
}
