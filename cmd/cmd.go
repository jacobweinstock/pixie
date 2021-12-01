package cmd

import (
	"context"
	"flag"
	"net/url"
	"os"
	"strings"

	"github.com/go-logr/logr"
	"github.com/go-logr/zerologr"
	"github.com/go-playground/validator/v10"
	"github.com/hashicorp/go-multierror"
	"github.com/jacobweinstock/ipxe"
	"github.com/jacobweinstock/proxydhcp/cli"
	"github.com/jacobweinstock/proxydhcp/proxy"
	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	"inet.af/netaddr"
)

const name = "pixie"

type config struct {
	LogLevel         string `validate:"oneof=debug info"`
	IPXEAddr         string `validate:"required,ip"`
	IPXEScriptAddr   string `validate:"required,url"`
	IPXEScript       string `validate:"required"`
	ProxyDHCPAddr    string `validate:"required,ip"`
	CustomUserClass  string
	DisableIPXE      bool
	DisableProxyDHCP bool
	Log              logr.Logger
}

func Execute(ctx context.Context) error {
	c := &config{}
	root := &ffcli.Command{
		Name:    name,
		FlagSet: registerFlags(c, name, flag.ExitOnError),
		Options: []ff.Option{
			ff.WithEnvVarPrefix(strings.ToUpper(name)),
		},
		Subcommands: []*ffcli.Command{
			cli.SupportedBins(ctx),
			file(&fileCfg{config: c}),
			tink(&tinkCfg{config: c}),
		},
		Exec: func(ctx context.Context, args []string) error {
			if err := validate(c); err != nil {
				return err
			}
			if err := validator.New().Struct(c); err != nil {
				return err
			}
			c.Log = defaultLogger(c.LogLevel)
			c.Log = c.Log.WithName("pixie")
			c.Log.Info("Starting")

			a := all{config: c}
			return a.exec(ctx, proxy.AllowAll{})
		},
	}
	return root.ParseAndRun(ctx, os.Args[1:])
}

func registerFlags(c *config, name string, errHandler flag.ErrorHandling) *flag.FlagSet {
	fs := flag.NewFlagSet(name, errHandler)
	fs.StringVar(&c.LogLevel, "log-level", "info", "log level")
	fs.StringVar(&c.IPXEAddr, "ipxe-addr", "", "address for servering tftp (port 69) and http (port 80) ipxe files")
	fs.StringVar(&c.IPXEScriptAddr, "ipxe-script-addr", "", "address that serves the ipxe script (http://192.168.2.2)")
	fs.StringVar(&c.IPXEScript, "ipxe-script-name", "auto.ipxe", "ipxe script name. used with ipxe-script-addr (http://192.168.2.2/<mac-addr>/auto.ipxe)")
	fs.StringVar(&c.ProxyDHCPAddr, "proxy-dhcp-addr", "", "address to listen on for proxy dhcp")
	fs.StringVar(&c.CustomUserClass, "custom-user-class", "", "custom user class")
	fs.BoolVar(&c.DisableIPXE, "disable-ipxe", false, "disable ipxe")
	fs.BoolVar(&c.DisableProxyDHCP, "disable-proxy-dhcp", false, "disable proxy dhcp")

	return fs
}

func validate(c *config) error {
	return validator.New().Struct(c)
}

// defaultLogger is a zerolog logr implementation.
func defaultLogger(level string) logr.Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	zerologr.NameFieldName = "logger"
	zerologr.NameSeparator = "/"

	zl := zerolog.New(os.Stdout)
	zl = zl.With().Caller().Timestamp().Logger()
	var l zerolog.Level
	switch level {
	case "debug":
		l = zerolog.DebugLevel
	default:
		l = zerolog.InfoLevel
	}
	zl = zl.Level(l)

	return zerologr.New(&zl)
}

type all struct {
	*config
	*fileCfg
	*tinkCfg
}

func (c all) exec(ctx context.Context, a proxy.Allower) error {
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
			return c.runProxyDHCP(ctx, a)
		})
	}
	if len(enabled) == 0 {
		c.Log.Info("No services enabled")
	}
	return g.Wait()
}

func (c *all) runProxyDHCP(ctx context.Context, a proxy.Allower) error {
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
		proxy.WithAllower(a),
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

func (c *all) runIPXE(ctx context.Context) error {
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
