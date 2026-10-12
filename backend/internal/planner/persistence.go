package planner

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func atomicWriteJSON(path string, value any, backup bool) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if backup {
		if old, err := os.ReadFile(path); err == nil {
			var x any
			if json.Unmarshal(old, &x) == nil {
				_ = os.WriteFile(path+".bak", old, 0o600)
			}
		}
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err = tmp.Write(raw); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = replaceFile(tmpName, path); err != nil {
		return err
	}
	return syncDir(filepath.Dir(path))
}

func loadJSONWithRecovery(path string) (map[string]any, string, error) {
	candidates := []string{path, path + ".bak", path + ".old"}
	matches, _ := filepath.Glob(path + ".*.tmp")
	sort.Sort(sort.Reverse(sort.StringSlice(matches)))
	if len(matches) > 3 {
		matches = matches[:3]
	}
	candidates = append(candidates, matches...)
	found := false
	var firstErr error
	for _, c := range candidates {
		f, err := os.Open(c)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		found = true
		b, err := io.ReadAll(f)
		_ = f.Close()
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		var v map[string]any
		if err := json.Unmarshal(b, &v); err == nil && v != nil {
			return v, c, nil
		} else if firstErr == nil {
			firstErr = err
		}
	}
	if found && firstErr != nil {
		return nil, "", fmt.Errorf("Planner JSON and recovery files are invalid for %s: %w", path, firstErr)
	}
	return nil, "", nil
}

func normalizeDataPath(raw string) (string, error) {
	v := strings.TrimSpace(os.ExpandEnv(raw))
	if strings.HasPrefix(v, "~") {
		home, _ := os.UserHomeDir()
		if v == "~" {
			v = home
		} else if strings.HasPrefix(v, "~/") || strings.HasPrefix(v, "~\\") {
			v = filepath.Join(home, v[2:])
		}
	}
	if v == "" {
		return "", fmt.Errorf("JSON 저장 위치를 입력하세요.")
	}
	if !filepath.IsAbs(v) {
		abs, err := filepath.Abs(v)
		if err != nil {
			return "", err
		}
		v = abs
	}
	if st, err := os.Stat(v); err == nil && st.IsDir() {
		v = filepath.Join(v, "eisenhower.json")
	} else if filepath.Ext(v) == "" {
		v = filepath.Join(v, "eisenhower.json")
	}
	if strings.ToLower(filepath.Ext(v)) != ".json" {
		return "", fmt.Errorf("저장 파일은 .json 확장자를 사용해야 합니다.")
	}
	return filepath.Clean(v), nil
}
