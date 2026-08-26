package cmd

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

func newProfileEditCmd(getDeps depsFunc) *cobra.Command {
	var (
		label              string
		baseURL            string
		defaultModel       string
		configDirFlag      string
		rotateCredential   bool
		credentialTypeFlag string
		apiKeyStdin        bool
	)

	cmd := &cobra.Command{
		Use:   "edit <name>",
		Short: "Edit an existing profile",
		Long: "Edit an existing profile. Pass at least one flag; --base-url and\n" +
			"--default-model accept an empty string to clear a previously set\n" +
			"value. --config-dir only repoints this profile's CLAUDE_CONFIG_DIR\n" +
			"in profiles.json, it never moves any files on disk.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			changedAnything := cmd.Flags().Changed("label") ||
				cmd.Flags().Changed("base-url") ||
				cmd.Flags().Changed("default-model") ||
				cmd.Flags().Changed("config-dir") ||
				rotateCredential ||
				credentialTypeFlag != ""

			if !changedAnything {
				return errors.New("nothing to edit, pass at least one flag, see --help")
			}

			d, err := getDeps()
			if err != nil {
				return err
			}

			existing, err := d.profiles.Get(name)
			if err != nil {
				return err
			}

			updated := existing

			if cmd.Flags().Changed("label") {
				updated.Label = label
			}
			if cmd.Flags().Changed("base-url") {
				updated.BaseURL = baseURL
			}
			if cmd.Flags().Changed("default-model") {
				updated.DefaultModel = defaultModel
			}
			if cmd.Flags().Changed("config-dir") {
				if configDirFlag == "" {
					return errors.New("config dir cannot be cleared, it's required for every profile, use \"profile remove\" to delete a profile instead")
				}
				absDir, err := filepath.Abs(configDirFlag)
				if err != nil {
					return fmt.Errorf("resolving %s: %w", configDirFlag, err)
				}
				updated.ConfigDir = absDir
			}

			credType := existing.CredentialType
			typeChanged := false
			if credentialTypeFlag != "" {
				parsed, err := parseCredentialTypeFlag(credentialTypeFlag)
				if err != nil {
					return err
				}
				typeChanged = parsed != existing.CredentialType
				credType = parsed
			}
			updated.CredentialType = credType

			if cmd.Flags().Changed("config-dir") {
				if err := checkConfigDirNotClaimed(d.profiles, updated.ConfigDir, name); err != nil {
					return err
				}
			}

			needsRotation := rotateCredential || typeChanged
			var previousSecret string
			var hadPreviousSecret bool

			if needsRotation {
				// Captured so a failed profiles.json write below can put
				// the keychain back the way it was, rather than leaving
				// the secret rotated while the profile still records the
				// old credential type, or vice versa.
				if v, err := d.secrets.Get(name); err == nil {
					previousSecret = v
					hadPreviousSecret = true
				}

				p := newPrompter(cmd)
				secretValue, err := resolveSecret(cmd, p, credType, updated.ConfigDir, apiKeyStdin)
				if err != nil {
					return err
				}
				if err := d.secrets.Set(name, secretValue); err != nil {
					return fmt.Errorf("saving the rotated secret to the keychain: %w", err)
				}
			}

			updated.UpdatedAt = time.Now().UTC()

			if err := d.profiles.Update(updated); err != nil {
				if needsRotation {
					if !hadPreviousSecret {
						return fmt.Errorf("updating the profile failed after the secret had already been rotated, and there was no previous secret to restore, run \"profile edit --rotate-credential\" again once this is fixed: %w", err)
					}
					if rbErr := d.secrets.Set(name, previousSecret); rbErr != nil {
						return fmt.Errorf("updating the profile failed (%v), and rolling back the credential rotation also failed (%v): the keychain secret for %q may no longer match its recorded credential type, run \"profile edit --rotate-credential\" again to fix it", err, rbErr, name)
					}
					return fmt.Errorf("updating the profile failed, rolled the keychain back to its previous secret: %w", err)
				}
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Updated profile %q.\n", name)
			return nil
		},
	}

	cmd.Flags().StringVar(&label, "label", "", "set a new label")
	cmd.Flags().StringVar(&baseURL, "base-url", "", "set a new base url override (pass an empty string to clear it)")
	cmd.Flags().StringVar(&defaultModel, "default-model", "", "set a new default model override (pass an empty string to clear it)")
	cmd.Flags().StringVar(&configDirFlag, "config-dir", "", "repoint this profile's CLAUDE_CONFIG_DIR (metadata only, does not move any files)")
	cmd.Flags().BoolVar(&rotateCredential, "rotate-credential", false, "replace this profile's secret with a newly entered one")
	cmd.Flags().StringVar(&credentialTypeFlag, "credential-type", "", `change the credential type ("api-key" or "oauth"), forces a rotation since the new type needs a differently shaped secret`)
	cmd.Flags().BoolVar(&apiKeyStdin, "api-key-stdin", false, "read the rotated secret from stdin instead of prompting")

	return cmd
}
