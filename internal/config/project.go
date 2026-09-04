package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

const ProjectConfigName = "talknote-cli.json"

type Project struct {
	Dir   string            `json:"-"`
	Notes map[string]string `json:"notes"`
	DMs   map[string]string `json:"dms"`
}

func FindProject(startDir, stopDir string) (*Project, error) {
	current, err := filepath.Abs(startDir)
	if err != nil {
		return nil, err
	}
	stop := ""
	if stopDir != "" {
		stop, err = filepath.Abs(stopDir)
		if err != nil {
			return nil, err
		}
	}
	for {
		if stop != "" && current == stop {
			return nil, nil
		}
		path := filepath.Join(current, ".config", ProjectConfigName)
		data, readErr := os.ReadFile(path)
		if readErr == nil {
			var project Project
			if err := json.Unmarshal(data, &project); err != nil {
				return nil, fmt.Errorf("parse %s: %w", path, err)
			}
			project.Dir = current
			if project.Notes == nil {
				project.Notes = map[string]string{}
			}
			if project.DMs == nil {
				project.DMs = map[string]string{}
			}
			return &project, nil
		}
		if !errors.Is(readErr, fs.ErrNotExist) {
			return nil, readErr
		}
		parent := filepath.Dir(current)
		if parent == current {
			return nil, nil
		}
		current = parent
	}
}

func (p *Project) NoteAliases() []string { return sortedKeys(p.Notes) }
func (p *Project) DMAliases() []string   { return sortedKeys(p.DMs) }

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
