package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LoadCascadingEnvFiles looks for and loads cascading environment files:
// .env -> .env.local -> .env.<profile> -> .env.<profile>.local
// Files loaded later override earlier ones.
func LoadCascadingEnvFiles(baseDir, profile string) map[string]string {
	envMap := make(map[string]string)

	var targetFiles []string
	targetFiles = append(targetFiles, filepath.Join(baseDir, ".env"))
	targetFiles = append(targetFiles, filepath.Join(baseDir, ".env.local"))

	if profile != "" {
		targetFiles = append(targetFiles, filepath.Join(baseDir, fmt.Sprintf(".env.%s", profile)))
		targetFiles = append(targetFiles, filepath.Join(baseDir, fmt.Sprintf(".env.%s.local", profile)))
	}

	for _, f := range targetFiles {
		parseEnvFile(f, envMap)
	}

	return envMap
}

func parseEnvFile(path string, target map[string]string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			k := strings.TrimSpace(parts[0])
			v := strings.TrimSpace(parts[1])
			// Strip matching quotes
			if (strings.HasPrefix(v, "\"") && strings.HasSuffix(v, "\"")) ||
				(strings.HasPrefix(v, "'") && strings.HasSuffix(v, "'")) {
				if len(v) >= 2 {
					v = v[1 : len(v)-1]
				}
			}
			target[k] = v
		}
	}
}
