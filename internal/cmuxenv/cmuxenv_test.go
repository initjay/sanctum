package cmuxenv

import "testing"

func TestDetectAbsent(t *testing.T) {
	t.Setenv("CMUX_WORKSPACE_ID", "")
	t.Setenv("CMUX_SURFACE_ID", "")
	t.Setenv("CMUX_TAB_ID", "")

	info, present := Detect()

	if present {
		t.Errorf("expected present to be false, got info %+v", info)
	}
}

func TestDetectPresent(t *testing.T) {
	t.Setenv("CMUX_WORKSPACE_ID", "workspace-123")
	t.Setenv("CMUX_SURFACE_ID", "surface-456")
	t.Setenv("CMUX_TAB_ID", "")

	info, present := Detect()

	if !present {
		t.Fatalf("expected present to be true")
	}
	if info.WorkspaceID != "workspace-123" || info.SurfaceID != "surface-456" {
		t.Errorf("unexpected info: %+v", info)
	}
}

func TestDetectFallsBackToTabID(t *testing.T) {
	t.Setenv("CMUX_WORKSPACE_ID", "")
	t.Setenv("CMUX_SURFACE_ID", "")
	t.Setenv("CMUX_TAB_ID", "tab-789")

	info, present := Detect()

	if !present {
		t.Fatalf("expected present to be true")
	}
	if info.SurfaceID != "tab-789" {
		t.Errorf("expected SurfaceID to fall back to CMUX_TAB_ID, got %q", info.SurfaceID)
	}
}
