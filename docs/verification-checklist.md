# Verification checklist

Most of sanctum is covered by `go test ./...`, including a race detector
pass (`go test -race ./...`). This document is for the parts that can't be,
because they depend on real OAuth logins, the real macOS Keychain, or a
real cmux install. Some of these I've already run myself during
development and note that below; the ones marked "not yet run" need to be
done by hand before I'd call this fully trustworthy day to day, since
they're the parts no test suite can stand in for.

## Already run during development

- Created an API key profile with `sanctum profile add`
  (`--non-interactive --credential-type api-key --api-key-stdin`) against
  the real Keychain, on a real machine, more than once across different
  branches. Confirmed the item lands under `security find-generic-password
  -s sanctum` and never under a Claude Code branded service name.
- Confirmed no secret value ever appears in `profiles.json`, only
  `masked_secret` style output in `profile list` / `profile show`.
- Ran `sanctum env <profile>` and `eval`'d it, confirmed the right vars
  were exported and the expected ones unset.
- Ran `sanctum shell <profile>` for real, confirmed it execs into the
  target shell (verified with `--shell /bin/echo` as a stand in, and with
  a real shell) rather than leaving a wrapper process behind.
- Ran `sanctum profile edit --rotate-credential`, `sanctum profile show`,
  `sanctum status`, and `sanctum profile remove` against a real profile
  end to end, confirmed the config dir was left in place after removal and
  the Keychain item was actually gone afterward.
- Confirmed `sanctum status` correctly reports `CMUX_WORKSPACE_ID` /
  `CMUX_SURFACE_ID` when they're present in the environment.
- Ran an adversarial hygiene check by hand: exported bogus
  `ANTHROPIC_API_KEY`, `ANTHROPIC_AUTH_TOKEN`, and `AWS_PROFILE` in the
  parent shell before `sanctum shell <profile>`, then checked `env` inside
  the spawned shell and confirmed only the expected vars survived.
- Confirmed cleanup after every one of the above: no leftover Keychain
  items, no leftover files outside the intended temp/config directories.

## Not yet run, needs a human

These need a real Claude account and a real browser, which isn't
something I can do from an automated session.

1. **Real `claude setup-token` login.** Run `sanctum profile add
   <name> --credential-type oauth` (or pick oauth interactively) end to
   end, complete the real browser login, and confirm sanctum's parser
   actually finds the token in the command's real output. If it doesn't,
   confirm the fallback (show the raw output, prompt to paste the token by
   hand) kicks in cleanly rather than silently failing. Either way, note
   what the real output looked like so `internal/setuptoken`'s parser can
   be tightened against real data instead of the best effort heuristic it
   currently has to rely on.
2. **Two profiles active at once, for real.** Pick two real accounts
   (ideally one API key profile and one oauth profile), run `sanctum
   shell` for each in two separate terminal panes at the same time, and
   run `claude` in both. Confirm each one authenticates as the expected
   account with no cross contamination, and that creating/using the oauth
   session in one pane didn't touch the other pane's Keychain login state.
3. **Real cmux panes.** Wire `sanctum shell <profile>` into an actual
   cmux pane's `command` field per `docs/cmux-integration.md`, and confirm
   the pane boots directly into the scoped shell, survives a terminal
   resize and Ctrl-C without leaving a zombie process behind, and reports
   the correct profile from `sanctum status` including the real
   `CMUX_SURFACE_ID` for that specific pane.
4. **The one time Keychain permission prompt.** The first time a freshly
   built `sanctum` binary reads a Keychain item it didn't just create in
   the same process (which is the normal case: `profile add` creates it,
   a later `sanctum shell`/`env` invocation reads it), macOS may show a
   one time "sanctum wants to access..." prompt since it's an unsigned
   binary. Confirm this is a one time thing per binary build, not
   something that reappears on every invocation, and note here if that
   turns out not to be true.
