package profile

// Env var names sanctum reads and writes when scoping a Claude Code
// session to a profile.
const (
	EnvClaudeConfigDir       = "CLAUDE_CONFIG_DIR"
	EnvAnthropicAPIKey       = "ANTHROPIC_API_KEY"
	EnvClaudeOAuthToken      = "CLAUDE_CODE_OAUTH_TOKEN"
	EnvAnthropicBaseURL      = "ANTHROPIC_BASE_URL"
	EnvAnthropicModel        = "ANTHROPIC_MODEL"
	EnvAnthropicDefaultModel = "ANTHROPIC_DEFAULT_MODEL"
	EnvSanctumProfile        = "SANCTUM_PROFILE"
)

// isolationSensitiveVars are vars that must never leak from the parent
// shell into a sanctum-scoped session, regardless of what a given profile
// configures, because Claude Code's own credential precedence would let
// any of them silently override or compete with what sanctum explicitly
// sets. Kept as a single exported-in-spirit slice so it stays easy to grep
// and extend if Anthropic adds another precedence-affecting var later.
var isolationSensitiveVars = []string{
	"ANTHROPIC_AUTH_TOKEN",
	"ANTHROPIC_PROFILE",
	"AWS_ACCESS_KEY_ID",
	"AWS_SECRET_ACCESS_KEY",
	"AWS_SESSION_TOKEN",
	"AWS_PROFILE",
	"AWS_BEARER_TOKEN_BEDROCK",
	"CLAUDE_CODE_USE_BEDROCK",
	"CLAUDE_CODE_USE_VERTEX",
	"ANTHROPIC_VERTEX_PROJECT_ID",
	"CLOUD_ML_REGION",
}

// BuildEnv returns the env vars a session scoped to p should have set,
// given the profile's resolved secret value.
func BuildEnv(p Profile, secretValue string) map[string]string {
	vars := map[string]string{
		EnvClaudeConfigDir: p.ConfigDir,
		EnvSanctumProfile:  p.Name,
	}

	switch p.CredentialType {
	case CredentialAPIKey:
		vars[EnvAnthropicAPIKey] = secretValue
	case CredentialOAuthToken:
		vars[EnvClaudeOAuthToken] = secretValue
	}

	if p.BaseURL != "" {
		vars[EnvAnthropicBaseURL] = p.BaseURL
	}

	if p.DefaultModel != "" {
		// Both var names are set to the same value defensively: the exact
		// semantic split between ANTHROPIC_MODEL and
		// ANTHROPIC_DEFAULT_MODEL isn't fully pinned down, so setting both
		// is the safe choice rather than guessing which one a given Claude
		// Code version actually reads.
		vars[EnvAnthropicModel] = p.DefaultModel
		vars[EnvAnthropicDefaultModel] = p.DefaultModel
	}

	return vars
}

// BuildUnsetList returns every env var name that must be explicitly unset
// so nothing already exported in the parent shell can leak into a
// sanctum-scoped session and override what BuildEnv sets.
func BuildUnsetList(p Profile) []string {
	unset := make([]string, 0, len(isolationSensitiveVars)+3)
	unset = append(unset, isolationSensitiveVars...)

	// Whichever credential var this profile isn't using must still be
	// unset, since Claude Code checks both and an inactive one, if left
	// over from the parent shell, could still outrank the one sanctum sets.
	switch p.CredentialType {
	case CredentialAPIKey:
		unset = append(unset, EnvClaudeOAuthToken)
	case CredentialOAuthToken:
		unset = append(unset, EnvAnthropicAPIKey)
	}

	if p.BaseURL == "" {
		unset = append(unset, EnvAnthropicBaseURL)
	}

	if p.DefaultModel == "" {
		unset = append(unset, EnvAnthropicModel, EnvAnthropicDefaultModel)
	}

	return unset
}
