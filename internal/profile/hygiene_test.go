package profile

import "testing"

func contains(list []string, item string) bool {
	for _, v := range list {
		if v == item {
			return true
		}
	}
	return false
}

func TestBuildEnvAPIKeyProfile(t *testing.T) {
	p := Profile{
		Name:           "work-acme",
		CredentialType: CredentialAPIKey,
		ConfigDir:      "/home/work-acme",
	}

	vars := BuildEnv(p, "sk-secret")

	if vars[EnvAnthropicAPIKey] != "sk-secret" {
		t.Errorf("expected ANTHROPIC_API_KEY to be set, got %q", vars[EnvAnthropicAPIKey])
	}
	if _, ok := vars[EnvClaudeOAuthToken]; ok {
		t.Errorf("did not expect CLAUDE_CODE_OAUTH_TOKEN to be set")
	}
	if vars[EnvClaudeConfigDir] != "/home/work-acme" {
		t.Errorf("expected CLAUDE_CONFIG_DIR to be set, got %q", vars[EnvClaudeConfigDir])
	}
	if vars[EnvSanctumProfile] != "work-acme" {
		t.Errorf("expected SANCTUM_PROFILE to be set, got %q", vars[EnvSanctumProfile])
	}
	if _, ok := vars[EnvAnthropicBaseURL]; ok {
		t.Errorf("did not expect ANTHROPIC_BASE_URL to be set when unconfigured")
	}
}

func TestBuildEnvOAuthProfile(t *testing.T) {
	p := Profile{
		Name:           "personal",
		CredentialType: CredentialOAuthToken,
		ConfigDir:      "/home/personal",
	}

	vars := BuildEnv(p, "oauth-secret")

	if vars[EnvClaudeOAuthToken] != "oauth-secret" {
		t.Errorf("expected CLAUDE_CODE_OAUTH_TOKEN to be set, got %q", vars[EnvClaudeOAuthToken])
	}
	if _, ok := vars[EnvAnthropicAPIKey]; ok {
		t.Errorf("did not expect ANTHROPIC_API_KEY to be set")
	}
}

func TestBuildEnvWithBaseURLAndModel(t *testing.T) {
	p := Profile{
		Name:           "custom",
		CredentialType: CredentialAPIKey,
		ConfigDir:      "/home/custom",
		BaseURL:        "https://gateway.example.com",
		DefaultModel:   "claude-sonnet-5",
	}

	vars := BuildEnv(p, "sk-secret")

	if vars[EnvAnthropicBaseURL] != "https://gateway.example.com" {
		t.Errorf("expected base url to be set, got %q", vars[EnvAnthropicBaseURL])
	}
	if vars[EnvAnthropicModel] != "claude-sonnet-5" {
		t.Errorf("expected ANTHROPIC_MODEL to be set, got %q", vars[EnvAnthropicModel])
	}
	if vars[EnvAnthropicDefaultModel] != "claude-sonnet-5" {
		t.Errorf("expected ANTHROPIC_DEFAULT_MODEL to be set, got %q", vars[EnvAnthropicDefaultModel])
	}
}

func TestBuildUnsetListExcludesActiveCredentialVar(t *testing.T) {
	apiKeyProfile := Profile{Name: "a", CredentialType: CredentialAPIKey}
	unset := BuildUnsetList(apiKeyProfile)

	if contains(unset, EnvAnthropicAPIKey) {
		t.Errorf("did not expect the active credential var (api key) in the unset list")
	}
	if !contains(unset, EnvClaudeOAuthToken) {
		t.Errorf("expected the inactive credential var (oauth token) in the unset list")
	}

	oauthProfile := Profile{Name: "b", CredentialType: CredentialOAuthToken}
	unset = BuildUnsetList(oauthProfile)

	if contains(unset, EnvClaudeOAuthToken) {
		t.Errorf("did not expect the active credential var (oauth token) in the unset list")
	}
	if !contains(unset, EnvAnthropicAPIKey) {
		t.Errorf("expected the inactive credential var (api key) in the unset list")
	}
}

func TestBuildUnsetListAlwaysIncludesSensitiveVars(t *testing.T) {
	p := Profile{Name: "a", CredentialType: CredentialAPIKey}
	unset := BuildUnsetList(p)

	for _, v := range isolationSensitiveVars {
		if !contains(unset, v) {
			t.Errorf("expected %q in the unset list", v)
		}
	}
}

func TestBuildUnsetListDropsBaseURLAndModelWhenConfigured(t *testing.T) {
	p := Profile{
		Name:           "custom",
		CredentialType: CredentialAPIKey,
		BaseURL:        "https://gateway.example.com",
		DefaultModel:   "claude-sonnet-5",
	}
	unset := BuildUnsetList(p)

	if contains(unset, EnvAnthropicBaseURL) {
		t.Errorf("did not expect ANTHROPIC_BASE_URL in the unset list when configured")
	}
	if contains(unset, EnvAnthropicModel) || contains(unset, EnvAnthropicDefaultModel) {
		t.Errorf("did not expect model vars in the unset list when configured")
	}
}

func TestBuildUnsetListIncludesBaseURLAndModelWhenNotConfigured(t *testing.T) {
	p := Profile{Name: "plain", CredentialType: CredentialAPIKey}
	unset := BuildUnsetList(p)

	if !contains(unset, EnvAnthropicBaseURL) {
		t.Errorf("expected ANTHROPIC_BASE_URL in the unset list when not configured")
	}
	if !contains(unset, EnvAnthropicModel) || !contains(unset, EnvAnthropicDefaultModel) {
		t.Errorf("expected model vars in the unset list when not configured")
	}
}
