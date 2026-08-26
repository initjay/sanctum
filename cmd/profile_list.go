package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// profileListEntry is what `profile list` prints, either as a table row or
// as JSON. It never carries the raw secret, only maskSecret's output.
type profileListEntry struct {
	Name           string `json:"name"`
	Label          string `json:"label"`
	CredentialType string `json:"credential_type"`
	MaskedSecret   string `json:"masked_secret"`
	ConfigDir      string `json:"config_dir"`
	BaseURL        string `json:"base_url,omitempty"`
}

func newProfileListCmd(getDeps depsFunc) *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all profiles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := getDeps()
			if err != nil {
				return err
			}

			profiles, err := d.profiles.Load()
			if err != nil {
				return err
			}

			sort.Slice(profiles, func(i, j int) bool { return profiles[i].Name < profiles[j].Name })

			entries := make([]profileListEntry, 0, len(profiles))
			for _, p := range profiles {
				masked := "(no secret found)"
				if value, err := d.secrets.Get(p.Name); err == nil {
					masked = maskSecret(value)
				}

				entries = append(entries, profileListEntry{
					Name:           p.Name,
					Label:          p.Label,
					CredentialType: string(p.CredentialType),
					MaskedSecret:   masked,
					ConfigDir:      p.ConfigDir,
					BaseURL:        p.BaseURL,
				})
			}

			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(entries)
			}

			return writeProfileTable(cmd.OutOrStdout(), entries)
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "print as JSON instead of a table")

	return cmd
}

func writeProfileTable(w io.Writer, entries []profileListEntry) error {
	if len(entries) == 0 {
		_, err := fmt.Fprintln(w, `no profiles yet, run "sanctum profile add <name>" to create one`)
		return err
	}

	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tLABEL\tTYPE\tSECRET\tCONFIG DIR")
	for _, e := range entries {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", e.Name, e.Label, e.CredentialType, e.MaskedSecret, e.ConfigDir)
	}

	return tw.Flush()
}
