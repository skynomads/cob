package main

import (
	"context"
	"log"

	"github.com/fsnotify/fsnotify"
	"github.com/jgillich/cob/pkg/artifact"
)

type devCmd struct {
}

func (r *devCmd) Run() error {
	ctx := context.Background()

	pool, err := getPool()
	if err != nil {
		return err
	}

	watcher, err := pool.GetWatcher()
	if err != nil {
		return err
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			log.Println("event:", event)
			if event.Has(fsnotify.Write) {
				log.Println("modified file:", event.Name)

				pkg, cont := pool.FindArtifact(event.Name)
				if pkg != nil {
					dep := pool.FindPackageDependants(pkg)
					if len(dep) > 0 {
						for _, cont := range dep {
							go func(cont *artifact.Image) {
								g := pool.BuildImageWithPackages(ctx, cont)
								if err := g.Wait(); err != nil {
									log.Println("error:", err)
								}
							}(cont)
						}
					} else {
						if err := pkg.Build(); err != nil {
							log.Println("error:", err)
						}
					}
				}
				if cont != nil {
					if err := cont.Build(); err != nil {
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
