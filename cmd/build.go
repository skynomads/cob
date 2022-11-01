package cmd

import (
	"context"
)

type buildCmd struct {
}

func (r *buildCmd) Run() error {
	ctx := context.Background()

	builder, err := getBuilder()
	if err != nil {
		return err
	}
	defer builder.Pool.Stop()

	group, _ := builder.Pool.GroupContext(ctx)
	for _, p := range builder.Packages {
		group.Submit(p.Build)
	}
	if err := group.Wait(); err != nil {
		return err
	}

	group, _ = builder.Pool.GroupContext(ctx)
	for _, i := range builder.Images {
		group.Submit(i.Build)
	}
	if err := group.Wait(); err != nil {
		return err
	}

	return nil
}
