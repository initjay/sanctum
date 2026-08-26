package cmd

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/initjay/sanctum/internal/profile"
)

func newEnvCmd(getDeps depsFunc) *cobra.Command {
	return &cobra.Command{
		Use:   "env <profile>",
		Short: "Print export statements that activate a profile in the current shell",
		Long: "Print export and unset statements for the given profile.\n" +
			"Meant to be sourced into a shell you already have open, for example:\n\n" +
			"  eval \"$(sanctum env work-acme)\"",
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

			return writeEnvScript(cmd.OutOrStdout(), resolved)
		},
	}
}

// writeEnvScript prints shell export and unset statements for resolved,
// and nothing else. Every human facing message elsewhere in sanctum goes
// to stderr instead, so that `eval "$(sanctum env <profile>)"` is safe.
func writeEnvScript(w io.Writer, resolved profile.ResolvedEnv) error {
	names := make([]string, 0, len(resolved.Vars))
	for name := range resolved.Vars {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		if _, err := fmt.Fprintf(w, "export %s=%s\n", name, shellQuote(resolved.Vars[name])); err != nil {
			return err
		}
	}

	unset := append([]string(nil), resolved.UnsetVars...)
	sort.Strings(unset)
	for _, name := range unset {
		if _, err := fmt.Fprintf(w, "unset %s\n", name); err != nil {
			return err
		}
	}

	return nil
}

// shellQuote wraps s in single quotes for safe use in a POSIX shell,
// escaping any embedded single quotes. Single quoting preserves every
// other character literally, so this is safe regardless of what a secret
// value happens to contain.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
