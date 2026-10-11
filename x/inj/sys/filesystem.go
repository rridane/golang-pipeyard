package sys

import (
	"github.com/rridane/golang-pipeyard/x/mod"
	"os"
)

type FileSystemProcessorInjector struct {
}

func (i *FileSystemProcessorInjector) Inject(stageName string) *mod.FileSystemProcessor {
	baseDir, _ := os.Getwd()
	return &mod.FileSystemProcessor{BaseDir: baseDir}
}
