package verify

import (
	"fmt"
	"os"
	"path/filepath"
)

// CleanCache removes all cached artifacts (JARs, class listings) from the cache directory.
// Returns the number of files removed and total bytes freed.
func CleanCache(cacheDir string) (int, int64, error) {
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		return 0, 0, nil
	}

	var count int
	var totalBytes int64

	err := filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			count++
			totalBytes += info.Size()
		}
		return nil
	})
	if err != nil {
		return 0, 0, fmt.Errorf("walking cache dir: %w", err)
	}

	if err := os.RemoveAll(cacheDir); err != nil {
		return 0, 0, fmt.Errorf("removing cache dir: %w", err)
	}

	return count, totalBytes, nil
}
