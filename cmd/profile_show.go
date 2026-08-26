package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"
)

// profileDetail is what `profile show` prints, either as labeled lines or
// as JSON. It never carries the raw secret, only maskSecret's output.
type profileDetail struct {
	Name            string `json:"name"`
	Label           string `json:"label"`
	CredentialType  string `json:"credential_type"`
	MaskedSecret    string `json:"masked_secret"`
	ConfigDir       string `json:"config_dir"`
	ConfigDirExists bool   `json:"config_dir_exists"`
	BaseURL         string `json:"base_url,omitempty"`
	DefaultModel    string `json:"default_model,omitempty"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

func newProfileShowCmd(getDeps depsFunc) *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "show <profile>",
		Short: "Show details for one profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := getDeps()
			if err != nil {
				return err
			}

			p, err := d.profiles.Get(args[0])
			if err != nil {
				return err
			}

			masked := "(no secret found)"
			if value, err := d.secrets.Get(p.Name); err == nil {
				masked = maskSecret(value)
			}

			_, statErr := os.Stat(p.ConfigDir)

			detail := profileDetail{
				Name:            p.Name,
				Label:           p.Label,
				CredentialType:  string(p.CredentialType),
				MaskedSecret:    masked,
				ConfigDir:       p.ConfigDir,
				ConfigDirExists: statErr == nil,
				BaseURL:         p.BaseURL,
				DefaultModel:    p.DefaultModel,
				CreatedAt:       p.CreatedAt.Format(time.RFC3339),
				UpdatedAt:       p.UpdatedAt.Format(time.RFC3339),
			}

			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(detail)
			}

			return writeProfileDetail(cmd.OutOrStdout(), detail)
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "print as JSON instead of plain text")

	return cmd
}

func writeProfileDetail(w io.Writer, d profileDetail) error {
	lines := []struct {
		label string
		value string
	}{
		{"Name", d.Name},
		{"Label", d.Label},
		{"Credential type", d.CredentialType},
		{"Secret", d.MaskedSecret},
		{"Config dir", d.ConfigDir},
		{"Config dir exists", fmt.Sprintf("%v", d.ConfigDirExists)},
		{"Base URL", d.BaseURL},
		{"Default model", d.DefaultModel},
		{"Created", d.CreatedAt},
		{"Updated", d.UpdatedAt},
	}

	for _, line := range lines {
		if line.value == "" {
			continue
		}
		if _, err := fmt.Fprintf(w, "%-18s%s\n", line.label+":", line.value); err != nil {
			return err
		}
	}

	return nil
}
