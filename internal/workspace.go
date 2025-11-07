package internal

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// WorkspaceInfo represents workspace information
type WorkspaceInfo struct {
	Hash string
	Path string
	Name string
}

// DetectWorkspaces detects all workspaces from workspaceStorage
func DetectWorkspaces(basePath string) (map[string]*WorkspaceInfo, error) {
	workspaceStorage := filepath.Join(basePath, "workspaceStorage")
	workspaces := make(map[string]*WorkspaceInfo)

	entries, err := os.ReadDir(workspaceStorage)
	if err != nil {
		return workspaces, nil // Return empty map if directory doesn't exist
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		hash := entry.Name()
		workspaceJSONPath := filepath.Join(workspaceStorage, hash, "workspace.json")

		info := &WorkspaceInfo{
			Hash: hash,
		}

		// Try to read workspace.json
		if data, err := os.ReadFile(workspaceJSONPath); err == nil {
			var workspaceData struct {
				Folder string `json:"folder"`
			}
			if err := json.Unmarshal(data, &workspaceData); err == nil {
				info.Path = workspaceData.Folder
				// Extract name from path
				if info.Path != "" {
					info.Name = filepath.Base(info.Path)
				}
			}
		}

		workspaces[hash] = info
	}

	return workspaces, nil
}

// AssociateComposerWithWorkspace attempts to associate a composer with a workspace
func AssociateComposerWithWorkspace(composerID string, contexts []*MessageContext, workspaces map[string]*WorkspaceInfo) string {
	// Try to get projectLayouts from context
	for _, ctx := range contexts {
		if ctx.ComposerID == composerID && len(ctx.ProjectLayouts) > 0 {
			for _, layout := range ctx.ProjectLayouts {
				// Try to match layout to workspace
				for hash, workspace := range workspaces {
					if workspace.Path != "" && layout == workspace.Path {
						return hash
					}
				}
			}
		}
	}

	return ""
}
