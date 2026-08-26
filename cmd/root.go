// Package cmd wires up the sanctum command line tree.
package cmd

import (
	"github.com/spf13/cobra"

	"github.com/initjay/sanctum/internal/profile"
	"github.com/initjay/sanctum/internal/secret"
	"github.com/initjay/sanctum/internal/xdg"
)

// deps bundles the stores every sanctum command needs. Commands take a
// depsFunc rather than calling newDeps directly, so tests can swap in fakes
// without ever touching the real Keychain or a real profiles.json.
type deps struct {
	profiles *profile.Store
	secrets  secret.Store
}

type depsFunc func() (deps, error)

func newDeps() (deps, error) {
	path, err := xdg.ProfilesFile()
	if err != nil {
		return deps{}, err
	}

	return deps{
		profiles: profile.NewStore(path),
		secrets:  secret.NewKeychainStore(),
	}, nil
}

// NewRootCmd builds the sanctum command tree, wired to the real Keychain
// and the real profiles.json.
func NewRootCmd() *cobra.Command {
	return newRootCmdWithDeps(newDeps)
}

func newRootCmdWithDeps(getDeps depsFunc) *cobra.Command {
	root := &cobra.Command{
		Use:           "sanctum",
		Short:         "Locks terminal sessions to specific Claude Code credentials",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(newEnvCmd(getDeps))
	root.AddCommand(newShellCmd(getDeps))
	root.AddCommand(newProfileCmd(getDeps))

	return root
}

// Execute runs the real sanctum CLI and returns any error it encountered.
func Execute() error {
	return NewRootCmd().Execute()
}
