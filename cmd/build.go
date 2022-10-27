package main

import (
	"context"
)

type buildCmd struct {
}

func (r *buildCmd) Run() error {
	ctx := context.Background()

	pool, err := getPool()
	if err != nil {
		return err
	}

	for _, i := range pool.Images {
		if err := pool.BuildImageWithPackages(ctx, i).Wait(); err != nil {
			return err
		}
	}

	return nil
}
