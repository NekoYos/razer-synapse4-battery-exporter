package synapse

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const defaultLogGlob = `Razer\RazerAppEngine\User Data\Logs\systray_systrayv*.log`

func DefaultLogPath() (string, error) {
	paths, err := DefaultLogPaths()
	if err != nil {
		return "", err
	}
	return paths[0], nil
}

func DefaultLogPaths() ([]string, error) {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return nil, errors.New("LOCALAPPDATA is not set")
	}

	matches, err := filepath.Glob(filepath.Join(localAppData, defaultLogGlob))
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no Razer Synapse systray logs found at %s", filepath.Join(localAppData, defaultLogGlob))
	}

	sort.Slice(matches, func(i, j int) bool {
		left, leftErr := os.Stat(matches[i])
		right, rightErr := os.Stat(matches[j])
		if leftErr != nil || rightErr != nil {
			return matches[i] < matches[j]
		}
		return left.ModTime().After(right.ModTime())
	})

	return matches, nil
}
