package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/initjay/sanctum/internal/cmuxenv"
	"github.com/initjay/sanctum/internal/profile"
)

// errNotActivated is returned when SANCTUM_PROFILE isn't set, so `status`
// exits non-zero and is script-checkable.
var errNotActivated = errors.New("not running inside a sanctum-activated shell")

type statusInfo struct {
	Active              bool   `json:"active"`
	ProfileName         string `json:"profile_name,omitempty"`
	Label               string `json:"label,omitempty"`
	CredentialType      string `json:"credential_type,omitempty"`
	MaskedSecret        string `json:"masked_secret,omitempty"`
	ConfigDir           string `json:"config_dir,omitempty"`
	ConfigDirMatchesEnv bool   `json:"config_dir_matches_env"`
	InsideCmux          bool   `json:"inside_cmux"`
	CmuxWorkspaceID     string `json:"cmux_workspace_id,omitempty"`
	CmuxSurfaceID       string `json:"cmux_surface_id,omitempty"`
}

func newStatusCmd(getDeps depsFunc) *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show which profile, if any, is active in this shell",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmuxInfo, insideCmux := cmuxenv.Detect()

			name := os.Getenv(profile.EnvSanctumProfile)
			if name == "" {
				if asJSON {
					info := statusInfo{InsideCmux: insideCmux, CmuxWorkspaceID: cmuxInfo.WorkspaceID, CmuxSurfaceID: cmuxInfo.SurfaceID}
					if err := json.NewEncoder(cmd.OutOrStdout()).Encode(info); err != nil {
						return err
					}
				}
				return errNotActivated
			}

			d, err := getDeps()
			if err != nil {
				return err
			}

			p, err := d.profiles.Get(name)
			if err != nil {
				return fmt.Errorf("SANCTUM_PROFILE is set to %q, but that profile no longer exists: %w", name, err)
			}

			info := statusInfo{
				Active:              true,
				ProfileName:         p.Name,
				Label:               p.Label,
				CredentialType:      string(p.CredentialType),
				MaskedSecret:        resolveMaskedSecret(d.secrets, p.Name),
				ConfigDir:           p.ConfigDir,
				ConfigDirMatchesEnv: os.Getenv(profile.EnvClaudeConfigDir) == p.ConfigDir,
				InsideCmux:          insideCmux,
				CmuxWorkspaceID:     cmuxInfo.WorkspaceID,
				CmuxSurfaceID:       cmuxInfo.SurfaceID,
			}

			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(info)
			}

			return writeStatus(cmd.OutOrStdout(), info)
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "print as JSON instead of plain text")

	return cmd
}

func writeStatus(w io.Writer, info statusInfo) error {
	lines := []struct{ label, value string }{
		{"Profile", info.ProfileName},
		{"Label", info.Label},
		{"Credential type", info.CredentialType},
		{"Secret", info.MaskedSecret},
		{"Config dir", info.ConfigDir},
	}

	for _, line := range lines {
		if line.value == "" {
			continue
		}
		if _, err := fmt.Fprintf(w, "%-18s%s\n", line.label+":", line.value); err != nil {
			return err
		}
	}

	if !info.ConfigDirMatchesEnv {
		if _, err := fmt.Fprintln(w, "warning: this shell's CLAUDE_CONFIG_DIR no longer matches what this profile currently resolves to, it may have been edited since this shell started"); err != nil {
			return err
		}
	}

	if info.InsideCmux {
		if _, err := fmt.Fprintf(w, "%-18s%s\n", "Cmux workspace:", info.CmuxWorkspaceID); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "%-18s%s\n", "Cmux surface:", info.CmuxSurfaceID); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintln(w, "Not running inside cmux (or its env vars aren't set)."); err != nil {
			return err
		}
	}

	return nil
}
