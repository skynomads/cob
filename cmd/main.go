package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"
	kongyaml "github.com/alecthomas/kong-yaml"
	"github.com/skynomads/cob/artifact"
)

var cli struct {
	Watch watchCmd `cmd:"dev" help:"Watch mode."`
	Build buildCmd `cmd:"dev" help:"Build packages and images."`

	ConfigFile string `help:"Cob config file." short:"f" type:"path" default:"cob.yaml"`

	SigningKey    string   `help:"Key to use for signing." type:"path"`
	KeyringAppend []string `help:"Path to extra keys to include in the keyring" type:"path"`

	RepositoryAppend []string `help:"Path to extra repositories to include" type:"path"`

	Env map[string]string `help:"Environment variables to set"`

	Package struct {
		Source    []string `help:"Package source paths." type:"path"`
		Target    string   `help:"Package target path." type:"path" default:"dist/packages"`
		PreBuild  string   `help:"Pre-build command."`
		PostBuild string   `help:"Post-build command."`
		Workspace string   `help:"Workspace path." type:"path"`
	} `embed:"" prefix:"package-"`
	Image struct {
		Source    []string          `help:"Image source paths." type:"path"`
		Target    string            `help:"Image target path." type:"path" default:"dist/images"`
		PreBuild  string            `help:"Pre-build command."`
		PostBuild string            `help:"Post-build command."`
		Ref       map[string]string `help:"Image refs."`
	} `embed:"" prefix:"image-"`
}

func Run() {
	ctx := kong.Parse(&cli,
		kong.Name("cob"),
		kong.UsageOnError(),
	)

	if f, err := os.Open(cli.ConfigFile); err == nil {
		resolver, err := kongyaml.Loader(f)
		ctx.FatalIfErrorf(err)

		ctx = kong.Parse(&cli,
			kong.Name("cob"),
			kong.UsageOnError(),
			kong.Resolvers(resolver),
		)
	}

	for k, v := range cli.Env {
		ctx.FatalIfErrorf(os.Setenv(k, v))
	}

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

func getBuilder() (*artifact.Builder, error) {
	packages := []*artifact.Package{}
	images := []*artifact.Image{}

	if paths, err := glob(cli.Package.Source); err != nil {
		return nil, err
	} else {
		for _, p := range paths {
			pkg, err := artifact.NewPackage(p, cli.Package.Target)
			pkg.SigningKey = cli.SigningKey
			pkg.PreBuild = cli.Package.PreBuild
			pkg.PostBuild = cli.Package.PostBuild
			pkg.Workspace = cli.Package.Workspace
			if err != nil {
				return nil, err
			}
			packages = append(packages, pkg)
		}
	}

	if paths, err := glob(cli.Image.Source); err != nil {
		return nil, err
	} else {
		for _, p := range paths {
			image, err := artifact.NewImage(p, cli.Image.Target)
			if err != nil {
				return nil, err
			}
			path, err := filepath.Abs(cli.Package.Target)
			if err != nil {
				return nil, err
			}
			image.PreBuild = cli.Image.PreBuild
			image.PostBuild = cli.Image.PostBuild
			image.Ref = cli.Image.Ref[strings.TrimSuffix(filepath.Base(p), ".yaml")]
			image.ExtraRepos = append(cli.RepositoryAppend, path)
			image.ExtraKeys = cli.KeyringAppend
			images = append(images, image)
		}
	}

	return artifact.NewBuilder(packages, images)
}
