package cmdutil

import (
	"context"
	"log/slog"

	"gitee.com/oschina/gitee-cli/internal/config"
	"gitee.com/oschina/gitee-cli/pkg/gitee"
	"gitee.com/oschina/gitee-cli/pkg/giteego"
	"gitee.com/oschina/gitee-cli/pkg/iostreams"
)

type Factory struct {
	IOStreams   *iostreams.IOStreams
	GiteeClient func() (*gitee.Client, error)
	GoAPIClient func(pathBase, service string) (*giteego.Client, error)
	Config      func() (string, error)
	NoTUI       bool
	Hostname    string
	Context     context.Context
}

func NewFactory(ios *iostreams.IOStreams) *Factory {
	f := &Factory{
		IOStreams: ios,
		Config:    config.Token,
		Context:   context.Background(),
	}
	f.GiteeClient = func() (*gitee.Client, error) {
		token, err := config.TokenForHost(f.Hostname)
		if err != nil {
			return nil, err
		}
		apiPrefix := config.APIPrefixForHost(f.Hostname)
		slog.Debug("creating Gitee client", "host", f.Hostname, "api_prefix", apiPrefix)
		return gitee.NewClient(token, gitee.WithBaseURL(apiPrefix)), nil
	}
	// GoAPIClient builds a client for a gitee-go service. pathBase is the segment
	// between the go-api host and /gitee-go (plugin repo, "sa", or ""), and
	// service is the service segment (giteego.ServiceIPipe, ...).
	f.GoAPIClient = func(pathBase, service string) (*giteego.Client, error) {
		token, err := config.TokenForHost(f.Hostname)
		if err != nil {
			return nil, err
		}
		base := config.GoAPIBaseURL(f.Hostname, pathBase)
		slog.Debug("creating Go API client", "host", f.Hostname, "base_url", base)
		return giteego.NewClient(token, giteego.WithBaseURL(base), giteego.WithServicePath(service)), nil
	}
	return f
}

func (f *Factory) IsTUI() bool {
	if f.NoTUI {
		return false
	}
	return config.TUIEnabled() && f.IOStreams.IsTerminal() && f.IOStreams.IsStdinTerminal()
}
