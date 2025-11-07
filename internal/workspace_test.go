package internal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/iksnae/cursor-session/testutil"
)

func TestDetectWorkspaces(t *testing.T) {
	tmpDir := testutil.CreateTempDir(t)
	basePath := filepath.Join(tmpDir, "User")

	tests := []struct {
		name      string
		setup     func()
		wantCount int
		wantErr   bool
	}{
		{
			name: "no workspace storage",
			setup: func() {
				// Don't create workspaceStorage
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "single workspace",
			setup: func() {
				testutil.CreateWorkspaceFixture(t, basePath, "workspace1")
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name: "multiple workspaces",
			setup: func() {
				testutil.CreateWorkspaceFixture(t, basePath, "workspace1")
				testutil.CreateWorkspaceFixture(t, basePath, "workspace2")
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name: "workspace with workspace.json",
			setup: func() {
				workspaceDir := testutil.CreateWorkspaceFixture(t, basePath, "workspace1")
				// Verify workspace.json was created
				workspaceJSONPath := filepath.Join(workspaceDir, "workspace.json")
				if _, err := os.Stat(workspaceJSONPath); os.IsNotExist(err) {
					t.Fatalf("workspace.json was not created")
				}
			},
			wantCount: 1,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up
			workspaceStorage := filepath.Join(basePath, "workspaceStorage")
			os.RemoveAll(workspaceStorage)
			tt.setup()

			workspaces, err := DetectWorkspaces(basePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("DetectWorkspaces() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(workspaces) != tt.wantCount {
				t.Errorf("DetectWorkspaces() returned %d workspaces, want %d", len(workspaces), tt.wantCount)
			}

			// Verify workspace structure
			for hash, workspace := range workspaces {
				if hash == "" {
					t.Error("Workspace hash should not be empty")
				}
				if workspace.Hash != hash {
					t.Errorf("Workspace.Hash = %q, want %q", workspace.Hash, hash)
				}
			}
		})
	}
}

func TestDetectWorkspaces_WithWorkspaceJSON(t *testing.T) {
	tmpDir := testutil.CreateTempDir(t)
	basePath := filepath.Join(tmpDir, "User")

	workspaceDir := testutil.CreateWorkspaceFixture(t, basePath, "workspace1")

	// Update workspace.json with custom data
	workspaceJSONPath := filepath.Join(workspaceDir, "workspace.json")
	workspaceData := map[string]interface{}{
		"folder": "/custom/path/to/workspace",
	}
	jsonData, _ := json.Marshal(workspaceData)
	if err := os.WriteFile(workspaceJSONPath, jsonData, 0644); err != nil {
		t.Fatalf("Failed to write workspace.json: %v", err)
	}

	workspaces, err := DetectWorkspaces(basePath)
	if err != nil {
		t.Fatalf("DetectWorkspaces() error = %v", err)
	}

	if len(workspaces) != 1 {
		t.Fatalf("DetectWorkspaces() returned %d workspaces, want 1", len(workspaces))
	}

	workspace := workspaces["workspace1"]
	if workspace.Path != "/custom/path/to/workspace" {
		t.Errorf("Workspace.Path = %q, want /custom/path/to/workspace", workspace.Path)
	}
	if workspace.Name != "workspace" {
		t.Errorf("Workspace.Name = %q, want workspace", workspace.Name)
	}
}

func TestAssociateComposerWithWorkspace(t *testing.T) {
	workspaces := map[string]*WorkspaceInfo{
		"workspace1": {
			Hash: "workspace1",
			Path: "/path/to/workspace1",
			Name: "workspace1",
		},
		"workspace2": {
			Hash: "workspace2",
			Path: "/path/to/workspace2",
			Name: "workspace2",
		},
	}

	tests := []struct {
		name           string
		composerID     string
		contexts       []*MessageContext
		wantWorkspace  string
	}{
		{
			name:          "no matching context",
			composerID:    "composer1",
			contexts:      []*MessageContext{},
			wantWorkspace: "",
		},
		{
			name:       "matching context",
			composerID: "composer1",
			contexts: []*MessageContext{
				{
					ComposerID:    "composer1",
					ProjectLayouts: []string{"/path/to/workspace1"},
				},
			},
			wantWorkspace: "workspace1",
		},
		{
			name:       "multiple contexts, first matches",
			composerID: "composer1",
			contexts: []*MessageContext{
				{
					ComposerID:    "composer1",
					ProjectLayouts: []string{"/path/to/workspace1"},
				},
				{
					ComposerID:    "composer2",
					ProjectLayouts: []string{"/path/to/workspace2"},
				},
			},
			wantWorkspace: "workspace1",
		},
		{
			name:       "context with no project layouts",
			composerID: "composer1",
			contexts: []*MessageContext{
				{
					ComposerID:     "composer1",
					ProjectLayouts: []string{},
				},
			},
			wantWorkspace: "",
		},
		{
			name:       "context with non-matching project layout",
			composerID: "composer1",
			contexts: []*MessageContext{
				{
					ComposerID:     "composer1",
					ProjectLayouts: []string{"/path/to/other"},
				},
			},
			wantWorkspace: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AssociateComposerWithWorkspace(tt.composerID, tt.contexts, workspaces)
			if got != tt.wantWorkspace {
				t.Errorf("AssociateComposerWithWorkspace() = %q, want %q", got, tt.wantWorkspace)
			}
		})
	}
}


