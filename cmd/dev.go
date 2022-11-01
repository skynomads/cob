package main

import (
	"context"
	"log"

	"github.com/fsnotify/fsnotify"
	"github.com/skynomads/cob/artifact"
)

type devCmd struct {
}

func (r *devCmd) Run() error {
	ctx := context.Background()

	build := buildCmd{}
	if err := build.Run(); err != nil {
		return err
	}

	builder, err := getBuilder()
	if err != nil {
		return err
	}
	defer builder.Pool.Stop()

	watcher, err := builder.GetWatcher()
	if err != nil {
		return err
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if event.Has(fsnotify.Write) {
				pkg, img := builder.FindArtifact(event.Name)
				if pkg != nil {
					dep := builder.FindPackageDependants(pkg)
					if len(dep) > 0 {
						for _, img := range dep {
							go func(img *artifact.Image) {
								g := builder.BuildImageWithPackages(ctx, img)
								if err := g.Wait(); err != nil {
									log.Println("error:", err)
								}
							}(img)
						}
					} else {
						if err := pkg.Build(); err != nil {
							log.Println("error:", err)
						}
					}
				}
				if img != nil {
					if err := img.Build(); err != nil {
						log.Println("error:", err)
					}
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			log.Println("fsnotify error:", err)
		}
	}
}
