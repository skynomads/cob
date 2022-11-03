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

	basePath, err := os.Getwd()
	ctx.FatalIfErrorf(err)
	if _, err := os.Open(cli.ConfigFile); err == nil {
		p, err := filepath.Abs(cli.ConfigFile)
		ctx.FatalIfErrorf(err)
		basePath = filepath.Dir(p)
	}

	for i, c := range cli.Package.Config {
		if !filepath.IsAbs(c) {
			cli.Package.Config[i] = filepath.Join(basePath, c)
		}
	}
	for i, c := range cli.KeyringAppend {
		if !filepath.IsAbs(c) {
			cli.KeyringAppend[i] = filepath.Join(basePath, c)
		}
	}
	for i, c := range cli.Image.Config {
		if !filepath.IsAbs(c) {
			cli.Image.Config[i] = filepath.Join(basePath, c)
		}
	}
	if !filepath.IsAbs(cli.Package.Source) {
		cli.Package.Source = filepath.Join(basePath, cli.Package.Source)
	}
	if !filepath.IsAbs(cli.Package.Target) {
		cli.Package.Target = filepath.Join(basePath, cli.Package.Target)
	}
	if !filepath.IsAbs(cli.SigningKey) {
		cli.SigningKey = filepath.Join(basePath, cli.SigningKey)
	}
	if !filepath.IsAbs(cli.Image.Target) {
		cli.Image.Target = filepath.Join(basePath, cli.Image.Target)
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
			options := []melange.Option{
				melange.WithConfig(p),
				melange.WithOutDir(cli.Package.Target),
				melange.WithSourceDir(filepath.Dir(p)),
			}
			if len(cli.SigningKey) > 0 {
				options = append(options, melange.WithSigningKey(cli.SigningKey))
			}
			if len(cli.Package.Workspace) > 0 {
				options = append(options, melange.WithWorkspaceDir(cli.Package.Workspace))
			}
			if len(cli.Package.Source) > 0 {
				options = append(options, melange.WithSourceDir(cli.Package.Source))
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
			target, err := filepath.Abs(cli.Package.Target)
			if err != nil {
				return nil, err
			}
			options := []apko.Option{
				apko.WithConfig(p),
				apko.WithExtraRepos(append(cli.RepositoryAppend, target)),
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
