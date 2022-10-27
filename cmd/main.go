package main

import (
	"os"
	"path/filepath"

	"github.com/alecthomas/kong"
	kongyaml "github.com/alecthomas/kong-yaml"
	"github.com/jgillich/cob/pkg/artifact"
)

var cli struct {
	Dev   devCmd   `cmd:"dev" help:"Development mode."`
	Build buildCmd `cmd:"dev" help:"Build packages and images."`

	ConfigFile string `help:"Cob config file." short:"f" env:"COB_CONFIG_FILE" type:"path"`

	Package struct {
		Source    []string `help:"Package source paths." env:"COB_PACKAGE_SOURCE" type:"path"`
		Target    string   `help:"Package target path." env:"COB_PACKAGE_TARGET" default:"dist/packages"`
		PreBuild  string   `help:"Pre-build command." env:"COB_PACKAGE_PREBUILD"`
		PostBuild string   `help:"Post-build command." env:"COB_PACKAGE_POSTBUILD"`
	} `embed:"" prefix:"package."`
	Image struct {
		Source    []string          `help:"Image source paths." env:"COB_IMAGE_SOURCE" type:"path"`
		Target    string            `help:"Image target path." env:"COB_IMAGE_TARGET" default:"dist/images"`
		PreBuild  string            `help:"Pre-build command." env:"COB_IMAGE_PREBUILD"`
		PostBuild string            `help:"Post-build command." env:"COB_IMAGE_POSTBUILD"`
		Ref       map[string]string `help:"Image refs." env:"COB_IMAGE_REF"`
	} `embed:"" prefix:"image."`
}

func main() {
	ctx := kong.Parse(&cli,
		kong.Name("cob"),
		kong.UsageOnError(),
	)

	f, err := os.Open(cli.ConfigFile)
	ctx.FatalIfErrorf(err)

	resolver, err := kongyaml.Loader(f)
	ctx.FatalIfErrorf(err)

	ctx = kong.Parse(&cli,
		kong.Name("cob"),
		kong.UsageOnError(),
		kong.Resolvers(resolver),
	)

	ctx.FatalIfErrorf(ctx.Run())
}

func glob(paths []string) ([]string, error) {
	matches := []string{}
	for _, p := range paths {
		m, err := filepath.Glob(p)
		if err != nil {
			return nil, err
		}
		matches = append(matches, m...)
	}
	return matches, nil
}

func getPool() (*artifact.Pool, error) {
	packages := []*artifact.Package{}
	images := []*artifact.Image{}

	if paths, err := glob(cli.Package.Source); err != nil {
		return nil, err
	} else {
		for _, p := range paths {
			pkg, err := artifact.NewPackage(p, cli.Package.Target)
			if err != nil {
				return nil, err
			}
			packages = append(packages, pkg)
		}
	}

	if paths, err := glob(cli.Package.Source); err != nil {
		return nil, err
	} else {
		for _, p := range paths {
			image, err := artifact.NewImage(p, cli.Image.Target)
			if err != nil {
				return nil, err
			}
			images = append(images, image)
		}
	}

	return artifact.NewPool(packages, images)
}
