package cmd

import (
	"os"
	"path/filepath"
	"strings"

	apko "chainguard.dev/apko/pkg/build"
	melange "chainguard.dev/melange/pkg/build"
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
		Config    []string `help:"Package config paths." type:"path"`
		Target    string   `help:"Package target path." type:"path" default:"dist/packages"`
		PreBuild  string   `help:"Pre-build command."`
		PostBuild string   `help:"Post-build command."`
		Workspace string   `help:"Workspace path." type:"path"`
		Source    string   `help:"Source path." type:"path"`
	} `embed:"" prefix:"package-"`
	Image struct {
		Config    []string          `help:"Image config paths." type:"path"`
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

	if paths, err := glob(cli.Package.Config); err != nil {
		return nil, err
	} else {
		for _, p := range paths {
			options := []melange.Option{}
			if len(cli.SigningKey) > 0 {
				options = append(options, melange.WithSigningKey(cli.SigningKey))
			}
			if len(cli.Package.Workspace) > 0 {
				melange.WithWorkspaceDir(cli.Package.Workspace)
			}
			pkg, err := artifact.NewPackage(p, cli.Package.Target, options)
			pkg.PreBuild = cli.Package.PreBuild
			pkg.PostBuild = cli.Package.PostBuild
			if err != nil {
				return nil, err
			}
			packages = append(packages, pkg)
		}
	}

	if paths, err := glob(cli.Image.Config); err != nil {
		return nil, err
	} else {
		for _, p := range paths {
			path, err := filepath.Abs(cli.Package.Target)
			if err != nil {
				return nil, err
			}
			options := []apko.Option{
				apko.WithExtraRepos(append(cli.RepositoryAppend, path)),
				apko.WithExtraKeys(cli.KeyringAppend),
			}
			image, err := artifact.NewImage(p, cli.Image.Target, options)
			if err != nil {
				return nil, err
			}
			image.PreBuild = cli.Image.PreBuild
			image.PostBuild = cli.Image.PostBuild
			image.Ref = cli.Image.Ref[strings.TrimSuffix(filepath.Base(p), ".yaml")]
			images = append(images, image)
		}
	}

	return artifact.NewBuilder(packages, images)
}
