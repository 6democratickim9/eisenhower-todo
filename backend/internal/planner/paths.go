package planner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	AppDirName       = "EisenDay"
	LegacyAppDirName = "EisenhowerTodo"
)

type ResolvedPaths struct{ ConfigDir, ConfigPath, JiraPath, DataPath, RunningPath string }

func userConfigHome() string {
	if runtime.GOOS == "windows" {
		if v := os.Getenv("APPDATA"); v != "" {
			return v
		}
		if v := os.Getenv("LOCALAPPDATA"); v != "" {
			return v
		}
		h, _ := os.UserHomeDir()
		return filepath.Join(h, "AppData", "Roaming")
	}
	if runtime.GOOS == "darwin" {
		h, _ := os.UserHomeDir()
		return filepath.Join(h, "Library", "Application Support")
	}
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return v
	}
	h, _ := os.UserHomeDir()
	return filepath.Join(h, ".config")
}
func appDir() string {
	if v := strings.TrimSpace(os.Getenv("EISEN_DAY_CONFIG_DIR")); v != "" {
		_ = os.MkdirAll(v, 0o755)
		return v
	}
	p := filepath.Join(userConfigHome(), AppDirName)
	_ = os.MkdirAll(p, 0o755)
	return p
}

func safeReadMap(path string) map[string]any {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var v map[string]any
	if json.Unmarshal(b, &v) != nil {
		return nil
	}
	return v
}

func ResolvePaths() (ResolvedPaths, error) {
	base := appDir()
	cfgPath := filepath.Join(base, "config.json")
	jiraPath := filepath.Join(base, "jira-settings.json")
	running := filepath.Join(base, "running.json")
	defaultData := filepath.Join(base, "data", "eisenhower.json")
	_ = os.MkdirAll(filepath.Dir(defaultData), 0o755)
	if cfg := safeReadMap(cfgPath); cfg != nil {
		if s, ok := cfg["dataPath"].(string); ok && strings.TrimSpace(s) != "" {
			if p, err := normalizeDataPath(s); err == nil {
				return ResolvedPaths{base, cfgPath, jiraPath, p, running}, nil
			}
		}
	}
	legacy := filepath.Join(userConfigHome(), LegacyAppDirName)
	if cfg := safeReadMap(filepath.Join(legacy, "config.json")); cfg != nil {
		if s, ok := cfg["dataPath"].(string); ok && strings.TrimSpace(s) != "" {
			if p, err := normalizeDataPath(s); err == nil {
				out := map[string]any{"dataPath": p, "displayName": "Minju", "timezone": "local", "autoSyncMinutes": 15}
				if x, ok := cfg["displayName"]; ok {
					out["displayName"] = x
				}
				if x, ok := cfg["timezone"]; ok {
					out["timezone"] = x
				}
				_ = atomicWriteJSON(cfgPath, out, false)
				oldJ := filepath.Join(legacy, "jira-settings.json")
				if _, err := os.Stat(oldJ); err == nil {
					if _, err2 := os.Stat(jiraPath); os.IsNotExist(err2) {
						if b, e := os.ReadFile(oldJ); e == nil {
							_ = os.WriteFile(jiraPath, b, 0o600)
						}
					}
				}
				return ResolvedPaths{base, cfgPath, jiraPath, p, running}, nil
			}
		}
	}
	return ResolvedPaths{base, cfgPath, jiraPath, defaultData, running}, nil
}
