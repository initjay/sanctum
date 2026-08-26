# sanctum

An isolated environment manager for your command line that locks individual terminal sessions to specific AI organization credentials. It dynamically scopes API keys and environment variables to a single window or pane, preventing accidental cross-contamination and unintended credit usage between projects.

Right now that means one thing concretely: managing multiple Claude Code CLI accounts so different terminal panes can each be pinned to a different account or org, without their credentials leaking into each other. macOS only for now.

## Building

Requires Go 1.25 or newer (see `go.mod`) and Xcode command line tools (for the macOS Keychain bindings).

```
go build -o sanctum .
```

## Usage

Everything lives under a "profile": a name, an isolated `CLAUDE_CONFIG_DIR`, and either an API key or a long lived OAuth token stored in the macOS Keychain under its own service name, separate from Claude Code's own Keychain items.

```
sanctum profile add work-acme        # create a profile, interactively or via flags
sanctum profile list                 # see every profile, secrets masked
sanctum profile show work-acme       # detail for one profile
sanctum profile edit work-acme       # change a label, base url, default model, or rotate the secret
sanctum profile remove work-acme     # delete a profile (never deletes its config dir)

sanctum shell work-acme              # launch a shell scoped to a profile
sanctum env work-acme                # print export/unset lines, for eval "$(sanctum env work-acme)"
sanctum status                       # show which profile, if any, is active in this shell
```

`sanctum shell` is the one meant to be wired into a terminal pane's startup command, so the pane boots directly into a scoped shell. See `docs/cmux-integration.md` for wiring it into [cmux](https://github.com/manaflow-ai/cmux) specifically.

`sanctum profile add <name> --credential-type oauth` runs `claude setup-token` for you, scoped to the new profile's isolated config dir, so a subscription based account can be isolated without ever touching the shared system Keychain login Claude Code otherwise uses.

See `docs/verification-checklist.md` for what's been tested and what still needs a manual pass.
