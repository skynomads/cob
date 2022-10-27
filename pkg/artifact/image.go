package artifact

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"sync"
	"time"

	"chainguard.dev/apko/pkg/build"
	"chainguard.dev/apko/pkg/build/oci"
	"chainguard.dev/apko/pkg/build/types"
	"gopkg.in/yaml.v3"
)

type Image struct {
	Source    string
	Target    string
	Ref       string
	PreBuild  string
	PostBuild string
	Config    types.ImageConfiguration
	lastBuild time.Time
	mutex     sync.Mutex
}

func NewImage(source string, target string) (*Image, error) {
	config, err := os.ReadFile(source)
	if err != nil {
		return nil, err
	}

	var ic types.ImageConfiguration
	if err := yaml.Unmarshal(config, &ic); err != nil {
		return nil, fmt.Errorf("failed to parse image configuration: %w", err)
	}

	return &Image{
		Source: source,
		Target: target,
		Config: ic,
	}, nil
}

func (i *Image) Build() error {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	i.lastBuild = time.Now()

	if len(i.PreBuild) > 0 {
		if _, err := exec.Command("/bin/sh", "-c", i.PreBuild).Output(); err != nil {
			return err
		}
	}

	wd, err := os.MkdirTemp("", "apko-*")
	if err != nil {
		return fmt.Errorf("failed to create working directory: %w", err)
	}
	defer os.RemoveAll(wd)

	bc, err := build.New(wd, build.WithConfig(i.Source))
	if err != nil {
		return err
	}

	if err := bc.Refresh(); err != nil {
		return err
	}

	// if bc.Options.SBOMPath == "" {
	// 	dir, err := filepath.Abs(outputTarGZ)
	// 	if err != nil {
	// 		return fmt.Errorf("resolving output file path: %w", err)
	// 	}
	// 	bc.Options.SBOMPath = filepath.Dir(dir)
	// }

	// if len(bc.ImageConfiguration.Archs) != 0 {
	// 	bc.Logger().Printf("WARNING: ignoring archs in config, only building for current arch (%s)", bc.Options.Arch)
	// }

	// bc.Logger().Printf("building image '%s'", imageRef)

	layer, err := bc.BuildLayer()
	if err != nil {
		return fmt.Errorf("failed to build layer image: %w", err)
	}
	defer os.Remove(layer)

	output := path.Join(i.Target, fmt.Sprintf("%s.tar.gz", path.Base(i.Target)))

	if err := oci.BuildImageTarballFromLayer(
		i.Ref, layer, output, bc.ImageConfiguration, bc.Options.SourceDateEpoch, bc.Options.Arch,
		bc.Logger(), bc.Options.SBOMPath, bc.Options.SBOMFormats); err != nil {
		return fmt.Errorf("failed to build OCI image: %w", err)
	}

	if len(i.PostBuild) > 0 {
		if _, err := exec.Command("/bin/sh", "-c", i.PostBuild).Output(); err != nil {
			return err
		}
	}

	return nil
}
