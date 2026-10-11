package sys

import (
	"io/fs"

	"github.com/rridane/golang-pipeyard/x/mod"
)

type AssetFinderInjector struct {
	FS fs.FS
}

func (d *AssetFinderInjector) Inject(stageName string) *mod.AssetFinder {
	if d.FS != nil {
		return &mod.AssetFinder{FS: d.FS}
	}
	return nil
	//return &mod.AssetFinder{FS: embeddedAssets}
}
