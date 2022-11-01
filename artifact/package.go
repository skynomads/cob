package artifact

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"chainguard.dev/apko/pkg/build/types"
	"chainguard.dev/melange/pkg/build"
	"gopkg.in/yaml.v3"
)

type Package struct {
	Source     string
	Target     string
	PreBuild   string
	PostBuild  string
	SigningKey string
	Config     build.Configuration
	lastBuild  time.Time
	mutex      sync.Mutex
}

func NewPackage(source string, target string) (*Package, error) {
	config, err := os.ReadFile(source)
	if err != nil {
		return nil, err
	}

	var pc build.Configuration
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

	options := []build.Option{
		build.WithConfig(p.Source),
		build.WithOutDir(p.Target),
		build.WithWorkspaceDir(""),
		build.WithArch(types.ParseArchitecture("amd64")),
		build.WithGenerateIndex(true),
		build.WithSigningKey(p.SigningKey),
	}

	bc, err := build.New(options...)
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
