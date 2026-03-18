package mod

import (
	"io"
	"os"
	"path/filepath"
)

type FileSystemProcessor struct {
}

func (f *FileSystemProcessor) CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

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

func (f *FileSystemProcessor) MergeFiles(pattern, output string) error {
	out, err := os.Create(output)
	if err != nil {
		return err
	}
	defer out.Close()

	files, err := filepath.Glob(pattern)
	if err != nil {
		return err
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

	return nil
}
