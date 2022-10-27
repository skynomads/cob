package artifact

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	melange "chainguard.dev/melange/pkg/build"
	"gopkg.in/yaml.v3"
)

type Package struct {
	Source    string
	Target    string
	PreBuild  string
	PostBuild string
	Config    melange.Configuration
	lastBuild time.Time
	mutex     sync.Mutex
}

func NewPackage(source string, target string) (*Package, error) {
	config, err := os.ReadFile(source)
	if err != nil {
		return nil, err
	}

	var pc melange.Configuration
	if err := yaml.Unmarshal(config, &pc); err != nil {
		return nil, fmt.Errorf("failed to parse package configuration: %w", err)
	}

	return &Package{
		Source: source,
		Target: target,
		Config: pc,
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

	options := []melange.Option{
		melange.WithConfig(p.Source),
		melange.WithOutDir(p.Target),
	}

	bc, err := melange.New(options...)
	if err != nil {
		return err
	}

	if err := bc.BuildPackage(); err != nil {
		return fmt.Errorf("failed to build package: %w", err)
	}

	if len(p.PostBuild) > 0 {
		if _, err := exec.Command("/bin/sh", "-c", p.PostBuild).Output(); err != nil {
			return err
		}
	}

	return nil
}
