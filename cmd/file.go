package cmd

import (
	"context"
	"flag"
	"net/url"

	"github.com/go-playground/validator/v10"
	ipxe "github.com/jacobweinstock/ipxe/cli"
	"github.com/jacobweinstock/proxydhcp/proxy"
	"github.com/peterbourgon/ff/v3/ffcli"
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
			cf := ipxe.NewFile(
				ipxe.WithLogger(c.Log),
				ipxe.WithFilename(c.Filename),
				ipxe.WithHTTP(c.IPXEAddr+":80"),
				ipxe.WithTFTPAddr(c.IPXEAddr+":69"),
				ipxe.WithLogLevel(c.LogLevel),
			)
			return cf.Exec(ctx, nil)
		})
	}
	if !c.DisableProxyDHCP {
		enabled = append(enabled, "proxy-dhcp")
		g.Go(func() error {
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
			opts := []proxy.Option{
				proxy.WithLogger(c.Log),
				proxy.WithTFTPAddr(ta),
				proxy.WithHTTPAddr(ha),
				proxy.WithIPXEAddr(ia),
			}
			if c.IPXEScript == "" {
				opts = append(opts, proxy.WithIPXEScript(c.IPXEScript))
			}
			if c.CustomUserClass != "" {
				opts = append(opts, proxy.WithUserClass(c.CustomUserClass))
			}
			h := proxy.NewHandler(ctx, opts...)

			rs, err := h.ServeRedirection(ctx, c.ProxyDHCPAddr)
			if err != nil {
				return err
			}

			bs, err := h.ServeBoot(ctx, c.ProxyDHCPAddr)
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
				rs.Close()
				bs.Close()
				return nil
			}
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
