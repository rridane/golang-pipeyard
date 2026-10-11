package filesystem

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

type FileSystemProcessor struct {
	BaseDir string
}

/* -------------------- Paths -------------------- */

func (f *FileSystemProcessor) ExecContextDir() string {
	return f.BaseDir
}

func (f *FileSystemProcessor) Abs(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(f.BaseDir, path)
}

/* -------------------- File ops -------------------- */

func (f *FileSystemProcessor) Exists(path string) bool {
	_, err := os.Stat(f.Abs(path))
	return err == nil
}

func (f *FileSystemProcessor) IsDir(path string) bool {
	info, err := os.Stat(f.Abs(path))
	if err != nil {
		return false
	}
	return info.IsDir()
}

func (f *FileSystemProcessor) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(f.Abs(path))
}

func (f *FileSystemProcessor) WriteFile(path string, data []byte) error {
	full := f.Abs(path)

	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		return err
	}

	return os.WriteFile(full, data, 0644)
}

/* -------------------- Copy -------------------- */

func (f *FileSystemProcessor) CopyFile(src, dst string) error {
	src = f.Abs(src)
	dst = f.Abs(dst)

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return out.Sync()
}

/* -------------------- Merge -------------------- */

func (f *FileSystemProcessor) MergeFiles(pattern, output string) error {
	output = f.Abs(output)

	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return err
	}

	out, err := os.Create(output)
	if err != nil {
		return err
	}
	defer out.Close()

	files, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return errors.New("no files matched pattern")
	}

	for _, file := range files {
		in, err := os.Open(file)
		if err != nil {
			return err
		}

		if _, err := io.Copy(out, in); err != nil {
			in.Close()
			return err
		}
		in.Close()
	}

	return out.Sync()
}

/* -------------------- Directory -------------------- */

func (f *FileSystemProcessor) ListFiles(dir string) ([]string, error) {
	dir = f.Abs(dir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, e := range entries {
		files = append(files, e.Name())
	}

	return files, nil
}

func (f *FileSystemProcessor) Walk(dir string, fn filepath.WalkFunc) error {
	return filepath.Walk(f.Abs(dir), fn)
}

/* -------------------- Delete -------------------- */

func (f *FileSystemProcessor) Remove(path string) error {
	return os.RemoveAll(f.Abs(path))
}
