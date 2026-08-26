package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newProfileRemoveCmd(getDeps depsFunc) *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a profile",
		Long: "Remove a profile's metadata and its stored secret. This never\n" +
			"deletes the profile's CLAUDE_CONFIG_DIR directory, it can hold\n" +
			"session history and settings worth keeping, so it's left on disk\n" +
			"and its path is printed for manual cleanup if you want it gone.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			d, err := getDeps()
			if err != nil {
				return err
			}

			p, err := d.profiles.Get(name)
			if err != nil {
				return err
			}

			if !yes {
				prompt := newPrompter(cmd)
				fmt.Fprintf(prompt.out, "This removes profile %q and its stored secret from the keychain. Its config dir at %s will not be deleted.\n", name, p.ConfigDir)
				answer, err := prompt.line("Continue? [y/N]: ")
				if err != nil {
					return err
				}
				if !strings.EqualFold(answer, "y") {
					return errors.New("aborted")
				}
			}

			// secret.Store.Delete already treats "there was nothing to
			// delete" as success, so any error here is a genuine failure
			// (a locked keychain, a denied access prompt), never just a
			// missing item. The profile metadata is still removed either
			// way, so a repeatedly failing keychain delete can never
			// leave behind a ghost profile with no way to remove it, but
			// that failure has to make the command exit non-zero, not
			// just print a warning, or a script checking the exit code
			// would have no way to notice the secret might still be
			// sitting in the keychain.
			secretErr := d.secrets.Delete(name)

			if err := d.profiles.Remove(name); err != nil {
				if secretErr != nil {
					return fmt.Errorf("removing the profile failed (%v), and its keychain secret could also not be deleted (%v)", err, secretErr)
				}
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Removed profile %q. Its config dir at %s was left in place.\n", name, p.ConfigDir)

			if secretErr != nil {
				return fmt.Errorf("the profile was removed, but its keychain secret could not be deleted and may still be present: %w", secretErr)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip the confirmation prompt")

	return cmd
}
