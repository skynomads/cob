package artifact

import (
	"context"
	"runtime"

	"github.com/alitto/pond"
	"github.com/fsnotify/fsnotify"
)

type Pool struct {
	Packages []*Package
	Images   []*Image
	pool     *pond.WorkerPool
}

func NewPool(packages []*Package, images []*Image) (*Pool, error) {
	pool := pond.New(runtime.NumCPU(), 1000)
	return &Pool{
		Packages: packages,
		Images:   images,
		pool:     pool,
	}, nil
}

func (p *Pool) BuildImageWithPackages(ctx context.Context, image *Image) *pond.TaskGroupWithContext {
	group, _ := p.pool.GroupContext(ctx)

	group.Submit(func() error {
		group, _ := p.pool.GroupContext(ctx)

		for _, dep := range p.FindImageDependencies(image) {
			group.Submit(func() error {
				return dep.Build()
			})
		}

		if err := group.Wait(); err != nil {
			return err
		}

		return image.Build()
	})

	return group
}

func (p *Pool) GetWatcher() (*fsnotify.Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	for _, p := range p.Packages {
		if err := watcher.Add(p.Source); err != nil {
			return nil, err
		}
	}

	for _, c := range p.Images {
		if err := watcher.Add(c.Source); err != nil {
			return nil, err
		}
	}

	return watcher, nil
}

func (p *Pool) FindImageDependencies(image *Image) []*Package {
	deps := []*Package{}
	for _, dep := range image.Config.Contents.Packages {
		for _, pkg := range p.Packages {
			if pkg.Config.Package.Name == dep {
				deps = append(deps, pkg)
			} else {
				for _, spkg := range pkg.Config.Subpackages {
					if spkg.Name == dep {
						deps = append(deps, pkg)
						break
					}
				}
			}
		}
	}
	return deps
}

func (p *Pool) FindPackageDependants(pkg *Package) []*Image {
	dep := []*Image{}
	for _, image := range p.Images {
		for _, cpkg := range image.Config.Contents.Packages {
			if cpkg == pkg.Config.Package.Name {
				dep = append(dep, image)
				break
			}
		}
	}
	return dep
}

func (p *Pool) FindArtifact(source string) (*Package, *Image) {
	for _, pkg := range p.Packages {
		if pkg.Source == source {
			return pkg, nil
		}
	}
	for _, image := range p.Images {
		if image.Source == source {
			return nil, image
		}
	}
	return nil, nil
}

func (p *Pool) Stop() {
	p.pool.Stop()
}

func (p *Pool) StopAndWait() {
	p.pool.StopAndWait()
}
