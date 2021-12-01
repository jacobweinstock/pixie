package cmd

import (
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"net/url"

	"github.com/go-playground/validator/v10"
	"github.com/hashicorp/go-multierror"
	"github.com/jacobweinstock/ipxe"
	pdFile "github.com/jacobweinstock/proxydhcp/authz/file"
	"github.com/jacobweinstock/proxydhcp/proxy"
	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/pkg/errors"
	"github.com/tinkerbell/tink/protos/hardware"
	"golang.org/x/sync/errgroup"
	"inet.af/netaddr"
)

const fileCLI = "file"

type fileCfg struct {
	*config
	Filename string `validate:"required"`
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

func (c *fileCfg) runProxyDHCP(ctx context.Context) error {
	ta, err := netaddr.ParseIPPort(c.IPXEAddr + ":69")
	if err != nil {
		return err
	}
	ha, err := netaddr.ParseIPPort(c.IPXEAddr + ":80")
	if err != nil {
		return err
	}
	ia, err := url.Parse(c.IPXEScriptAddr)
	if err != nil {
		return err
	}

	saData, err := ioutil.ReadFile(c.Filename)
	if err != nil {
		return errors.Wrapf(err, "could not read file %q", c.Filename)
	}
	dsDB := []*hardware.Hardware{}
	if err := json.Unmarshal(saData, &dsDB); err != nil {
		return errors.Wrapf(err, "unable to parse configuration file %q", c.Filename)
	}

	fb := &pdFile.File{DB: dsDB}

	opts := []proxy.Option{
		proxy.WithLogger(c.Log),
		proxy.WithTFTPAddr(ta),
		proxy.WithHTTPAddr(ha),
		proxy.WithIPXEAddr(ia),
		proxy.WithAllower(fb),
	}
	if c.IPXEScript == "" {
		opts = append(opts, proxy.WithIPXEScript(c.IPXEScript))
	}
	if c.CustomUserClass != "" {
		opts = append(opts, proxy.WithUserClass(c.CustomUserClass))
	}
	h := proxy.NewHandler(ctx, opts...)

	u, err := netaddr.ParseIPPort(c.ProxyDHCPAddr + ":67")
	if err != nil {
		return err
	}
	rs, err := h.Server(u)
	if err != nil {
		return err
	}

	h2 := proxy.NewHandler(ctx, opts...)
	bs, err := h2.Server(u.WithPort(4011))
	if err != nil {
		return err
	}

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		h.Log.Info("starting proxydhcp", "addr1", c.ProxyDHCPAddr, "addr2", "0.0.0.0:67")
		return rs.Serve()
	})
	g.Go(func() error {
		h.Log.Info("starting proxydhcp", "addr1", c.ProxyDHCPAddr, "addr2", "0.0.0.0:4011")
		return bs.Serve()
	})

	errCh := make(chan error)
	go func() {
		errCh <- g.Wait()
	}()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		h.Log.Info("shutting down")
		return multierror.Append(nil, rs.Close(), bs.Close()).ErrorOrNil()
	}
}

func (c *fileCfg) runIPXE(ctx context.Context) error {
	hAddr, err := netaddr.ParseIPPort(c.IPXEAddr + ":80")
	if err != nil {
		return err
	}
	tAddr, err := netaddr.ParseIPPort(c.IPXEAddr + ":69")
	if err != nil {
		return err
	}
	cf := ipxe.Config{
		TFTP: ipxe.TFTP{
			Addr: tAddr,
		},
		HTTP: ipxe.HTTP{
			Addr: hAddr,
		},
		MACPrefix: true,
		Log:       c.Log,
	}

	return cf.Serve(ctx)
}

func (c *fileCfg) exec(ctx context.Context) error {
	if err := validator.New().Struct(c); err != nil {
		return err
	}
	c.Log = defaultLogger(c.LogLevel)
	c.Log = c.Log.WithName("pixie")
	c.Log.Info("Starting")

	var enabled []string
	g, ctx := errgroup.WithContext(ctx)
	if !c.DisableIPXE {
		enabled = append(enabled, "ipxe")
		g.Go(func() error {
			return c.runIPXE(ctx)
		})
	}
	if !c.DisableProxyDHCP {
		enabled = append(enabled, "proxy-dhcp")
		g.Go(func() error {
			return c.runProxyDHCP(ctx)
		})
	}
	if len(enabled) == 0 {
		c.Log.Info("No services enabled")
	}
	return g.Wait()
}

func (c *fileCfg) registerFlags(name string, errHandler flag.ErrorHandling) *flag.FlagSet {
	fs := registerFlags(c.config, name, errHandler)
	fs.StringVar(&c.Filename, "filename", "", "filename")
	return fs
}
