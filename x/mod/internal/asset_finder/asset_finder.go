package asset_finder

import (
	"crypto/sha1"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

type AssetFinder struct {
	FS       fs.FS
	cacheDir string
	once     sync.Once
}

func (a *AssetFinder) ensureCacheDir() error {
	var err error
	a.once.Do(func() {
		var dir string
		dir, err = os.MkdirTemp("", "pipeyard-assets-*")
		if err != nil {
			return
		}
		a.cacheDir = dir
	})
	return err
}

func (a *AssetFinder) ReadFile(name string) ([]byte, error) {
	return fs.ReadFile(a.FS, name)
}

func (a *AssetFinder) Exists(name string) bool {
	_, err := fs.Stat(a.FS, name)
	return err == nil
}

func (a *AssetFinder) Path(name string) (string, error) {
	if err := a.ensureCacheDir(); err != nil {
		return "", err
	}

	data, err := fs.ReadFile(a.FS, name)
	if err != nil {
		return "", err
	}

	sum := sha1.Sum([]byte(name))
	filename := hex.EncodeToString(sum[:]) + "-" + filepath.Base(name)
	target := filepath.Join(a.cacheDir, filename)

	if _, err := os.Stat(target); err == nil {
		return target, nil
	}

	if err := os.WriteFile(target, data, 0o644); err != nil {
		return "", err
	}

	return target, nil
}

func (a *AssetFinder) ListRecursive(dir string) ([]string, error) {
	var files []string

	err := fs.WalkDir(a.FS, dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

func (a *AssetFinder) List(dir string) ([]string, error) {
	entries, err := fs.ReadDir(a.FS, dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		files = append(files, entry.Name())
	}

	return files, nil
}
