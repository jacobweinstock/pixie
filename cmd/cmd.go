package cmd

import (
	"context"
	"os"

	"github.com/jacobweinstock/pixie/cli"
)

func Execute(ctx context.Context) error {
	root := cli.Pixie(ctx)
	return root.ParseAndRun(ctx, os.Args[1:])
}
