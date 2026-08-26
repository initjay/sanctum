package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/initjay/sanctum/internal/profile"
	"github.com/initjay/sanctum/internal/setuptoken"
	"github.com/initjay/sanctum/internal/xdg"
)

func newProfileAddCmd(getDeps depsFunc) *cobra.Command {
	var (
		label              string
		credentialTypeFlag string
		apiKeyStdin        bool
		baseURL            string
		defaultModel       string
		configDirFlag      string
		reuseConfigDir     bool
		nonInteractive     bool
	)

	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Create a new profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if !profile.ValidName(name) {
				return fmt.Errorf("invalid profile name %q, use only letters, numbers, hyphens, and underscores", name)
			}

			d, err := getDeps()
			if err != nil {
				return err
			}

			if _, err := d.profiles.Get(name); err == nil {
				return fmt.Errorf("a profile named %q already exists", name)
			} else if !errors.Is(err, profile.ErrNotFound) {
				return err
			}

			p := newPrompter(cmd)

			resolvedLabel := label
			if resolvedLabel == "" && !nonInteractive {
				resolvedLabel, err = p.line("Label (optional, press enter to skip): ")
				if err != nil {
					return err
				}
			}

			credType, err := resolveCredentialType(p, credentialTypeFlag, nonInteractive)
			if err != nil {
				return err
			}

			if credType == profile.CredentialOAuthToken && nonInteractive {
				return errors.New("oauth credentials require an interactive browser login, --non-interactive isn't supported with --credential-type oauth")
			}

			configDir, err := resolveConfigDir(p, configDirFlag, name, reuseConfigDir, nonInteractive)
			if err != nil {
				return err
			}

			secretValue, err := resolveSecret(cmd, p, credType, configDir, apiKeyStdin)
			if err != nil {
				return err
			}

			resolvedBaseURL := baseURL
			if resolvedBaseURL == "" && !nonInteractive {
				resolvedBaseURL, err = p.line("Base URL override (optional, press enter to skip): ")
				if err != nil {
					return err
				}
			}

			resolvedModel := defaultModel
			if resolvedModel == "" && !nonInteractive {
				resolvedModel, err = p.line("Default model override (optional, press enter to skip): ")
				if err != nil {
					return err
				}
			}

			now := time.Now().UTC()
			newProfile := profile.Profile{
				Name:           name,
				Label:          resolvedLabel,
				CredentialType: credType,
				ConfigDir:      configDir,
				BaseURL:        resolvedBaseURL,
				DefaultModel:   resolvedModel,
				CreatedAt:      now,
				UpdatedAt:      now,
			}

			if err := d.profiles.Add(newProfile); err != nil {
				return err
			}

			if err := d.secrets.Set(name, secretValue); err != nil {
				// Roll back the profile metadata we just wrote, so a
				// failed secret save never leaves behind a profile with
				// no secret and no way to fix it until profile edit or
				// remove exist.
				_ = d.profiles.Remove(name)
				return fmt.Errorf("saving the secret to the keychain: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Created profile %q. Run \"sanctum shell %s\" to use it.\n", name, name)
			return nil
		},
	}

	cmd.Flags().StringVar(&label, "label", "", "a free text label for your own reference")
	cmd.Flags().StringVar(&credentialTypeFlag, "credential-type", "", `credential type: "api-key" or "oauth"`)
	cmd.Flags().BoolVar(&apiKeyStdin, "api-key-stdin", false, "read the secret from stdin instead of prompting")
	cmd.Flags().StringVar(&baseURL, "base-url", "", "override ANTHROPIC_BASE_URL for this profile")
	cmd.Flags().StringVar(&defaultModel, "default-model", "", "override the default model for this profile")
	cmd.Flags().StringVar(&configDirFlag, "config-dir", "", "isolated CLAUDE_CONFIG_DIR to use (defaults to a directory under sanctum's own config dir)")
	cmd.Flags().BoolVar(&reuseConfigDir, "reuse-config-dir", false, "reuse an existing, possibly non-empty config dir without prompting")
	cmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "fail instead of prompting for anything not supplied by a flag")

	return cmd
}

// prompter reads interactive answers from a single shared buffered reader
// over the command's stdin, so multiple prompts in one command invocation
// don't each create their own buffer and risk losing already-buffered
// input out from under each other.
type prompter struct {
	in  *bufio.Reader
	out io.Writer
}

func newPrompter(cmd *cobra.Command) *prompter {
	return &prompter{in: bufio.NewReader(cmd.InOrStdin()), out: cmd.OutOrStdout()}
}

func (p *prompter) line(prompt string) (string, error) {
	fmt.Fprint(p.out, prompt)

	text, err := p.in.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}

	return strings.TrimSpace(text), nil
}

func resolveCredentialType(p *prompter, flagValue string, nonInteractive bool) (profile.CredentialType, error) {
	if flagValue != "" {
		return parseCredentialTypeFlag(flagValue)
	}
	if nonInteractive {
		return "", errors.New("--credential-type is required with --non-interactive")
	}

	for {
		choice, err := p.line("Credential type - [1] API key (Console/org token) or [2] OAuth (subscription account, via 'claude setup-token'): ")
		if err != nil {
			return "", err
		}

		switch choice {
		case "1":
			return profile.CredentialAPIKey, nil
		case "2":
			return profile.CredentialOAuthToken, nil
		default:
			fmt.Fprintln(p.out, "please enter 1 or 2")
		}
	}
}

func parseCredentialTypeFlag(s string) (profile.CredentialType, error) {
	switch s {
	case "api-key":
		return profile.CredentialAPIKey, nil
	case "oauth":
		return profile.CredentialOAuthToken, nil
	default:
		return "", fmt.Errorf(`unknown credential type %q, expected "api-key" or "oauth"`, s)
	}
}

func resolveConfigDir(p *prompter, flagValue, name string, reuse, nonInteractive bool) (string, error) {
	configDir := flagValue
	if configDir == "" {
		homesDir, err := xdg.ClaudeHomesDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(homesDir, name)
	}

	entries, statErr := os.ReadDir(configDir)
	switch {
	case os.IsNotExist(statErr):
		if err := os.MkdirAll(configDir, 0o700); err != nil {
			return "", fmt.Errorf("creating %s: %w", configDir, err)
		}
	case statErr != nil:
		return "", statErr
	case len(entries) > 0 && !reuse:
		if nonInteractive {
			return "", fmt.Errorf("%s already exists and isn't empty, pass --reuse-config-dir to use it anyway", configDir)
		}

		answer, err := p.line(fmt.Sprintf("%s already exists and isn't empty. Use it anyway? [y/N]: ", configDir))
		if err != nil {
			return "", err
		}
		if !strings.EqualFold(answer, "y") {
			return "", fmt.Errorf("aborted, %s already exists", configDir)
		}
	}

	return configDir, nil
}

func resolveSecret(cmd *cobra.Command, p *prompter, credType profile.CredentialType, configDir string, apiKeyStdin bool) (string, error) {
	switch credType {
	case profile.CredentialAPIKey:
		return readSecretValue(p, apiKeyStdin, "API key")
	case profile.CredentialOAuthToken:
		return runSetupToken(cmd, p, configDir)
	default:
		return "", fmt.Errorf("unknown credential type %q", credType)
	}
}

// readSecretValue prompts for a secret, masking the input when stdin is a
// real terminal and reading a plain line otherwise, which is also what
// makes --api-key-stdin and piped/test input work: whenever forceStdin is
// set or stdin isn't a terminal, this always takes the same plain-line
// path a test can drive without needing a real TTY.
func readSecretValue(p *prompter, forceStdin bool, label string) (string, error) {
	if !forceStdin && term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Fprintf(p.out, "%s (input hidden): ", label)
		raw, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(p.out)
		if err != nil {
			return "", err
		}

		value := strings.TrimSpace(string(raw))
		if value == "" {
			return "", fmt.Errorf("%s cannot be empty", label)
		}
		return value, nil
	}

	value, err := p.line(fmt.Sprintf("%s: ", label))
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", fmt.Errorf("%s cannot be empty", label)
	}

	return value, nil
}

func runSetupToken(cmd *cobra.Command, p *prompter, configDir string) (string, error) {
	fmt.Fprintln(p.out, `Running "claude setup-token", follow the prompts to log in with the account this profile should use.`)

	result, err := setuptoken.Run(cmd.Context(), configDir, cmd.InOrStdin(), p.out)
	if err != nil {
		return "", err
	}

	if result.Token != "" {
		return result.Token, nil
	}

	fmt.Fprintln(p.out, "Could not automatically find the token in that output, paste it manually below.")
	return readSecretValue(p, false, "OAuth token")
}
