package version

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const fileName = "VERSION"

func Load() (string, error) {
	for _, candidate := range candidateFiles() {
		data, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}

		value := strings.TrimSpace(string(data))
		if value == "" {
			return "", fmt.Errorf("version file %s is empty", candidate)
		}

		return value, nil
	}

	return "", fmt.Errorf("version file %s not found", fileName)
}

func candidateFiles() []string {
	seen := make(map[string]struct{})
	candidates := make([]string, 0, 16)

	addWalk := func(start string) {
		if start == "" {
			return
		}

		dir := filepath.Clean(start)
		for {
			candidate := filepath.Join(dir, fileName)
			if _, ok := seen[candidate]; !ok {
				seen[candidate] = struct{}{}
				candidates = append(candidates, candidate)
			}

			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}

			dir = parent
		}
	}

	if cwd, err := os.Getwd(); err == nil {
		addWalk(cwd)
	}

	if executablePath, err := os.Executable(); err == nil {
		addWalk(filepath.Dir(executablePath))
	}

	return candidates
}
