package atlas

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

func DefaultIndexPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "cmdatlas", "index.json"), nil
}

func Load(path string) (Index, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Index{Version: CurrentIndexVersion}, nil
		}
		return Index{}, err
	}

	var index Index
	if err := json.Unmarshal(data, &index); err != nil {
		return Index{}, err
	}
	if index.Version == 0 {
		index.Version = CurrentIndexVersion
	}
	return index, nil
}

func Save(path string, index Index) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
