// Package execshell replaces the current process with a shell scoped to a
// resolved profile's environment.
package execshell

import (
	"os/exec"
	"strings"
	"syscall"
)

// BuildEnv overlays vars onto base (a slice of "KEY=VALUE" strings, the
// shape os.Environ() returns), dropping any existing entry for a name in
// unset or a name vars is about to set. The result never contains a stale
// or duplicate entry for any name sanctum controls, regardless of what was
// already present in base.
func BuildEnv(base []string, vars map[string]string, unset []string) []string {
	drop := make(map[string]bool, len(unset)+len(vars))
	for _, name := range unset {
		drop[name] = true
	}
	for name := range vars {
		drop[name] = true
	}

	env := make([]string, 0, len(base)+len(vars))
	for _, kv := range base {
		name, _, ok := strings.Cut(kv, "=")
		if ok && drop[name] {
			continue
		}
		env = append(env, kv)
	}

	for name, value := range vars {
		env = append(env, name+"="+value)
	}

	return env
}

// Exec replaces the current process image with shellPath, using env as its
// full environment. On success this never returns, the calling process
// becomes the shell. It's implemented with syscall.Exec rather than
// os/exec's Command+Wait so the spawned shell becomes a terminal pane's
// actual foreground process, with correct signal handling and no leftover
// wrapper process for the pane's pty to contend with.
func Exec(shellPath string, args []string, env []string) error {
	resolved, err := exec.LookPath(shellPath)
	if err != nil {
		return err
	}

	return syscall.Exec(resolved, args, env)
}
