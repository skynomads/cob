package artifact

import (
	"context"
	"path/filepath"
	"runtime"

	"chainguard.dev/apko/pkg/build/types"
	"chainguard.dev/melange/pkg/build"
	"chainguard.dev/melange/pkg/index"
	"github.com/alitto/pond"
	"github.com/fsnotify/fsnotify"
)

type Builder struct {
	Packages []*Package
	Images   []*Image
	Pool     *pond.WorkerPool
}

func NewBuilder(packages []*Package, images []*Image) (*Builder, error) {
	pool := pond.New(runtime.NumCPU(), 1000)
	return &Builder{
		Packages: packages,
		Images:   images,
		Pool:     pool,
	}, nil
}

func (b *Builder) BuildImageWithPackages(ctx context.Context, image *Image) *pond.TaskGroupWithContext {
	group, _ := b.Pool.GroupContext(ctx)

	group.Submit(func() error {
		group, _ := b.Pool.GroupContext(ctx)

		for _, dep := range b.FindImageDependencies(image) {
			group.Submit(func() error {
				return dep.Build()
			})
		}

		if err := group.Wait(); err != nil {
			return err
		}

		if err := b.GenerateIndex(); err != nil {
			return err
		}

		return image.Build()
	})

	return group
}

func (b *Builder) GetWatcher() (*fsnotify.Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	for _, p := range b.Packages {
		if err := watcher.Add(p.Source); err != nil {
			return nil, err
		}
	}

	for _, c := range b.Images {
		if err := watcher.Add(c.Source); err != nil {
			return nil, err
		}
	}

	return watcher, nil
}

func (b *Builder) FindImageDependencies(image *Image) []*Package {
	deps := []*Package{}
	for _, dep := range image.Config.Contents.Packages {
		for _, pkg := range b.Packages {
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

func (b *Builder) FindPackageDependants(pkg *Package) []*Image {
	dep := []*Image{}
	for _, image := range b.Images {
		for _, cpkg := range image.Config.Contents.Packages {
			if cpkg == pkg.Config.Package.Name {
				dep = append(dep, image)
				break
			}
		}
	}
	return dep
}

func (b *Builder) FindArtifact(source string) (*Package, *Image) {
	for _, pkg := range b.Packages {
		if pkg.Source == source {
			return pkg, nil
		}
	}
	for _, image := range b.Images {
		if image.Source == source {
			return nil, image
		}
	}
	return nil, nil
}

// TODO refactor
func (b *Builder) GenerateIndex() error {
	if len(b.Packages) == 0 {
		return nil
	}

	for _, arch := range types.ParseArchitectures(b.Packages[0].Config.Package.TargetArchitecture) {
		packageDir := filepath.Join(b.Packages[0].Target, arch.ToAPK())

		bc, err := build.New(b.Packages[0].Options...)
		if err != nil {
			return err
		}

		opts := []index.Option{
			index.WithPackageDir(packageDir),
			index.WithIndexFile(filepath.Join(packageDir, "APKINDEX.tar.gz")),
			index.WithSigningKey(bc.SigningKey),
		}

		ctx, err := index.New(opts...)
		if err != nil {
			return err
		}

		if err := ctx.GenerateIndex(); err != nil {
			return err
		}
	}

	return nil
}
