package cmd

import (
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"

	"github.com/go-playground/validator/v10"
	pdFile "github.com/jacobweinstock/proxydhcp/authz/file"
	"github.com/jacobweinstock/proxydhcp/proxy"
	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/pkg/errors"
	"github.com/tinkerbell/tink/protos/hardware"
)

const fileCLI = "file"

type fileCfg struct {
	*config
	Filename string `validate:"required"`
	authz    proxy.Allower
}

func file(c *fileCfg) *ffcli.Command {
	return &ffcli.Command{
		Name:       fileCLI,
		ShortUsage: fileCLI,
		FlagSet:    c.registerFlags(fileCLI, flag.ExitOnError),
		Exec: func(ctx context.Context, _ []string) error {
			return c.exec(ctx)
		},
	}
}

func (c *fileCfg) exec(ctx context.Context) error {
	if err := validator.New().Struct(c); err != nil {
		return err
	}
	c.Log = defaultLogger(c.LogLevel)
	c.Log = c.Log.WithName("pixie")
	c.Log.Info("Starting")

	a := all{config: c.config, fileCfg: c}
	saData, err := ioutil.ReadFile(c.Filename)
	if err != nil {
		return errors.Wrapf(err, "could not read file %q", c.Filename)
	}
	dsDB := []*hardware.Hardware{}
	if err := json.Unmarshal(saData, &dsDB); err != nil {
		return errors.Wrapf(err, "unable to parse configuration file %q", c.Filename)
	}

	fb := &pdFile.File{DB: dsDB}
	return a.exec(ctx, fb)
}

func (c *fileCfg) registerFlags(name string, errHandler flag.ErrorHandling) *flag.FlagSet {
	fs := registerFlags(c.config, name, errHandler)
	fs.StringVar(&c.Filename, "filename", "", "filename")
	return fs
}
