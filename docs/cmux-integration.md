# Wiring sanctum into cmux

sanctum doesn't read or write cmux's own config, and doesn't need to know
anything about cmux to work. The whole integration is one thing: point a
pane's startup command at `sanctum shell <profile-name>` so the pane boots
directly into a scoped shell instead of a bare one. Everything below is
about how to do that in cmux specifically, confirmed against cmux's actual
schema and docs rather than guessed at.

## The easy way: capture a layout, then edit it

The simplest path doesn't involve hand writing JSON at all:

1. Open the panes you want (one per profile), arrange them however you
   like.
2. Right click the plus button and choose **Save Workspace as Layout**.
   This captures the current splits, directories, and running commands
   into `~/.config/cmux/cmux.json`.
3. Open that file (cmux also has a **Customize Workspace Layouts** action
   that opens it directly) and add a `"command"` field to each surface you
   want scoped, set to `sanctum shell <profile-name>`.

That's it. Re-selecting that saved workspace from then on will boot every
surface straight into its scoped shell.

## The manual way: editing cmux.json directly

If you'd rather write the config by hand, here's the shape, based on
cmux's actual JSON schema and docs (not the top level `"workspaces"` key
some earlier, less careful notes assumed, that key doesn't exist).

cmux looks for config in three places, and merges them per entry (a
project local entry overrides a global one with the same name, entries
that only exist globally still apply, it isn't a single file replacing the
others wholesale):

- `./.cmux/cmux.json` (project local, takes precedence for a given entry)
- `./cmux.json` (legacy fallback)
- `~/.config/cmux/cmux.json` (global)

A workspace with a saved layout lives inside a `commands` array entry (or
an `actions` entry), under a `workspace` key, not at the file's top level.
A surface (one pane/tab) inside that layout supports a `command` field,
which is the shell command cmux runs the instant that surface is created.
Here's a two profile example, one pane per account, side by side. This
assumes `work-acme` and `personal` already exist as sanctum profiles
(`sanctum profile add <name>`), sanctum shell will just fail with "profile
not found" for a name that hasn't been created yet:

```json
{
  "commands": [
    {
      "name": "sanctum-accounts",
      "workspace": {
        "name": "Accounts",
        "layout": {
          "direction": "horizontal",
          "split": 0.5,
          "children": [
            {
              "pane": {
                "surfaces": [
                  { "type": "terminal", "name": "work-acme", "command": "sanctum shell work-acme", "focus": true }
                ]
              }
            },
            {
              "pane": {
                "surfaces": [
                  { "type": "terminal", "name": "personal", "command": "sanctum shell personal" }
                ]
              }
            }
          ]
        }
      }
    }
  ]
}
```

Run that command from cmux's command palette (or however you trigger a
saved `commands` entry) and you get two panes, each already running
`sanctum shell` for a different profile.

A couple of things worth knowing about `command` specifically:

- It's a single shell string, not an argv list. cmux sends it to the
  pane's shell the same way as if you'd typed it and pressed enter, full
  shell syntax works.
- Since `sanctum shell` replaces its own process with the target shell
  (`syscall.Exec`, not a child process), it fits the same pattern cmux's
  own docs use for handing a pane off to a long running program, their
  own examples end a `command` string with `exec codex --yolo` or
  `exec "${SHELL:-/bin/zsh}" -l` for exactly this reason. `sanctum shell`
  does that same handoff internally, so `"command": "sanctum shell
  work-acme"` is enough on its own, no `exec` prefix needed.
- There's no documented handling of the command's exit code, cmux appears
  to just attach the pane to whatever ends up running there.

## What sanctum reads from cmux

Purely for display, `sanctum status` reads `CMUX_WORKSPACE_ID` and
`CMUX_SURFACE_ID` (cmux injects these into every pane's environment) so
you can confirm which specific pane a profile is active in. Nothing else
in sanctum touches cmux's environment or config, and sanctum never writes
to `cmux.json`.
