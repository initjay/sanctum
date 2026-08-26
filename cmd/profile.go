package cmd

import "github.com/spf13/cobra"

func newProfileCmd(getDeps depsFunc) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Manage sanctum credential profiles",
	}

	cmd.AddCommand(newProfileListCmd(getDeps))
	cmd.AddCommand(newProfileShowCmd(getDeps))
	cmd.AddCommand(newProfileAddCmd(getDeps))
	cmd.AddCommand(newProfileEditCmd(getDeps))
	cmd.AddCommand(newProfileRemoveCmd(getDeps))

	return cmd
}
