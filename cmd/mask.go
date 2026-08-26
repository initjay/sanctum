package cmd

import "strings"

// maskSecret returns a display safe representation of a secret: a fixed
// run of asterisks, so the output doesn't even leak the secret's actual
// length, followed by its last 4 characters, enough for a user to
// recognize which credential they're looking at without exposing it.
func maskSecret(value string) string {
	const visible = 4
	const maskRun = "******"

	if len(value) <= visible {
		return strings.Repeat("*", len(value))
	}

	return maskRun + value[len(value)-visible:]
}
