package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/initjay/sanctum/internal/execshell"
	"github.com/initjay/sanctum/internal/profile"
)

func newShellCmd(getDeps depsFunc) *cobra.Command {
	var shellOverride string

	cmd := &cobra.Command{
		Use:   "shell <profile>",
		Short: "Launch a shell scoped to a profile",
		Long: "Launch a new shell with the given profile's credentials and\n" +
			"CLAUDE_CONFIG_DIR already set. This is meant to be wired into a\n" +
			"terminal pane's startup command so the pane boots directly into\n" +
			"the scoped environment.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := getDeps()
			if err != nil {
				return err
			}

			resolved, err := profile.ResolveProfile(d.profiles, d.secrets, args[0])
			if err != nil {
				return err
			}

			shellPath := resolveShellPath(shellOverride)
			env := execshell.BuildEnv(os.Environ(), resolved.Vars, resolved.UnsetVars)

			fmt.Fprintf(cmd.ErrOrStderr(), "sanctum: activating profile %q (%s)\n", resolved.ProfileName, shellPath)

			return execshell.Exec(shellPath, []string{filepath.Base(shellPath)}, env)
		},
	}

	cmd.Flags().StringVar(&shellOverride, "shell", "", "shell binary to launch instead of $SHELL")

	return cmd
}

// resolveShellPath picks which shell binary to launch: an explicit
// override, then $SHELL, then /bin/zsh as a last resort.
func resolveShellPath(override string) string {
	if override != "" {
		return override
	}
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}
	return "/bin/zsh"
}
