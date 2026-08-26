// Package cmuxenv reads the environment variables cmux injects into every
// terminal surface it creates. It's read only: sanctum never reads or
// writes cmux's own config, this is purely for display in `sanctum
// status` so a user can see which pane a profile is active in.
package cmuxenv

import "os"

// Info is what cmux tells a process about where it's running.
type Info struct {
	WorkspaceID string
	SurfaceID   string
}

// Detect reads cmux's env vars from the current process's environment.
// present reports whether either was actually set, so callers can tell
// "not running inside cmux" apart from a coincidentally empty value.
func Detect() (info Info, present bool) {
	info.WorkspaceID = os.Getenv("CMUX_WORKSPACE_ID")

	info.SurfaceID = os.Getenv("CMUX_SURFACE_ID")
	if info.SurfaceID == "" {
		info.SurfaceID = os.Getenv("CMUX_TAB_ID")
	}

	present = info.WorkspaceID != "" || info.SurfaceID != ""
	return info, present
}
