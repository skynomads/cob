package artifact

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"chainguard.dev/apko/pkg/build/types"
	"chainguard.dev/melange/pkg/build"
	"gopkg.in/yaml.v3"
)

type Package struct {
	Source    string
	Target    string
	PreBuild  string
	PostBuild string
	Config    build.Configuration
	Options   []build.Option
	lastBuild time.Time
	mutex     sync.Mutex
}

func NewPackage(source string, target string, options []build.Option) (*Package, error) {
	config, err := os.ReadFile(source)
	if err != nil {
		return nil, err
	}

	var pc build.Configuration
	if err := yaml.Unmarshal(config, &pc); err != nil {
		return nil, fmt.Errorf("failed to parse package configuration: %w", err)
	}

	arch := types.ParseArchitectures(pc.Package.TargetArchitecture)[0].ToAPK()
	var lastBuild time.Time
	if fi, err := os.Stat(filepath.Join(target, arch, fmt.Sprintf("%s-%s-r%d.apk", pc.Package.Name, pc.Package.Version, pc.Package.Epoch))); err == nil {
		lastBuild = fi.ModTime()
	}

	return &Package{
		Source:    source,
		Target:    target,
		Config:    pc,
		Options:   options,
		lastBuild: lastBuild,
	}, nil
}

func (p *Package) Build() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	fi, err := os.Stat(p.Source)
	if err != nil {
		return err
	}

	if fi.ModTime().Before(p.lastBuild) {
		return nil
	}

	p.lastBuild = time.Now()

	if len(p.PreBuild) > 0 {
		if _, err := exec.Command("/bin/sh", "-c", p.PreBuild).Output(); err != nil {
			return err
		}
	}

	for _, arch := range types.ParseArchitectures(p.Config.Package.TargetArchitecture) {
		options := append([]build.Option{
			build.WithConfig(p.Source),
			build.WithOutDir(p.Target),
			build.WithSourceDir(filepath.Dir(p.Source)),
			build.WithArch(arch),
			build.WithGenerateIndex(false),
		}, p.Options...)

		bc, err := build.New(options...)
		if err != nil {
			return err
		}

		if err := bc.BuildPackage(); err != nil {
			return fmt.Errorf("failed to build package: %w", err)
		}
	}

	if len(p.PostBuild) > 0 {
		if _, err := exec.Command("/bin/sh", "-c", p.PostBuild).Output(); err != nil {
			return err
		}
	}

	return nil
}
