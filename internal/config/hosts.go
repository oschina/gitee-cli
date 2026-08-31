package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type HostConfig struct {
	Token     string `yaml:"token"`
	APIPrefix string `yaml:"api_prefix"`
}

func hostsPath() string {
	return fmt.Sprintf("%s/hosts.yml", ConfigDir())
}

func hostsViper() *viper.Viper {
	v := viper.NewWithOptions(viper.KeyDelimiter("::"))
	v.SetConfigType("yaml")
	v.SetConfigFile(hostsPath())
	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) && !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "warning: failed to read hosts config: %v\n", err)
		}
	}
	return v
}

func ListHosts() []string {
	v := hostsViper()
	settings := v.AllSettings()
	hosts := make([]string, 0, len(settings))
	for h := range settings {
		hosts = append(hosts, h)
	}
	return hosts
}

// DefaultHostname returns the configured host, falling back to the only saved
// private host when gitee.com has no credentials.
func DefaultHostname() string {
	hostname := Get(KeyHost)
	if hostname != "" && hostname != DefaultHost {
		return hostname
	}
	if _, err := Token(); err == nil {
		return DefaultHost
	}
	hosts := ListHosts()
	if len(hosts) == 1 {
		return hosts[0]
	}
	return DefaultHost
}

func GetHostConfig(hostname string) (HostConfig, bool) {
	v := hostsViper()
	if !v.IsSet(hostname) {
		return HostConfig{}, false
	}
	return HostConfig{
		Token:     v.GetString(hostname + "::token"),
		APIPrefix: v.GetString(hostname + "::api_prefix"),
	}, true
}

func SaveHostConfig(hostname, token, apiPrefix string) error {
	if err := os.MkdirAll(ConfigDir(), 0700); err != nil {
		return fmt.Errorf("config: create dir: %w", err)
	}
	v := hostsViper()
	v.Set(hostname+"::token", token)
	if apiPrefix != "" {
		v.Set(hostname+"::api_prefix", apiPrefix)
	} else {
		v.Set(hostname+"::api_prefix", "https://"+hostname+"/api/v5")
	}
	return writeYAMLAtomic(hostsPath(), v.AllSettings(), 0600)
}

func DeleteHostConfig(hostname string) error {
	v := hostsViper()
	if !v.IsSet(hostname) {
		return fmt.Errorf("host %q not found", hostname)
	}
	settings := v.AllSettings()
	delete(settings, hostname)

	return writeYAMLAtomic(hostsPath(), settings, 0600)
}

func TokenForHost(hostname string) (string, error) {
	if hostname == "" || hostname == DefaultHost {
		return Token()
	}
	hc, ok := GetHostConfig(hostname)
	if !ok || hc.Token == "" {
		return "", fmt.Errorf("not logged in to %s: run `gitee auth login --hostname %s`", hostname, hostname)
	}
	return hc.Token, nil
}

func APIPrefixForHost(hostname string) string {
	if hostname == "" || hostname == DefaultHost {
		return APIPrefix()
	}
	hc, ok := GetHostConfig(hostname)
	if ok && hc.APIPrefix != "" {
		return hc.APIPrefix
	}
	return "https://" + hostname + "/api/v5"
}

// GoAPIHost returns the gitee-go API host for the given Gitee host.
// Precedence: explicit config `go_api_host` > `gitee.com` special-case
// (go-api.gitee.com) > `*.runjs.cn` special-case (local-pipe-api.runjs.cn)
// > same host as the Gitee instance.
func GoAPIHost(giteeHostname string) string {
	if h, ok := isExplicitGoAPIHost(); ok {
		return h
	}
	if giteeHostname == "" || giteeHostname == DefaultHost {
		return DefaultGoAPIHost
	}
	if strings.HasSuffix(giteeHostname, ".runjs.cn") {
		return LocalGoAPIHost
	}
	return giteeHostname
}

// GoAPIBaseURL returns the gitee-go gateway base URL for a repo path segment
// (the resolved owner/repo) between the host and /gitee-go:
//
//	https://{go-api-host}/{pathBase}/gitee-go              (public gitee.com / explicit go_api_host / *.runjs.cn)
//	https://{gitee-host}/go-api/{pathBase}/gitee-go        (premium/private: fixed /go-api segment)
//
// For the public gitee.com the host is already go-api.gitee.com, so no extra
// /go-api path segment appears; for a non-gitee.com Gitee hostname the gateway
// is reached via a fixed /go-api segment on the same host. An explicitly
// configured go_api_host is used verbatim as the host (no /go-api segment).
// The *.runjs.cn special-case resolves to local-pipe-api.runjs.cn (no /go-api
// segment).
// The service segment (ipipe, sa, ...) and its versioned path are appended by
// the caller, since they differ per gitee-go service.
func GoAPIBaseURL(giteeHostname, pathBase string) string {
	host := GoAPIHost(giteeHostname)
	// An explicitly configured go_api_host already points at a go-api host;
	// the gitee.com and *.runjs.cn special-cases resolve to their go-api
	// hosts. Only the same-host fallback (premium/private deployments) needs
	// the fixed /go-api path segment.
	_, explicit := isExplicitGoAPIHost()
	if !explicit && host != DefaultGoAPIHost && host != LocalGoAPIHost {
		// same-host fallback: route through /go-api on the Gitee host
		return "https://" + host + "/go-api" + joinPath(pathBase) + GoAPIBasePath
	}
	return "https://" + host + joinPath(pathBase) + GoAPIBasePath
}

func joinPath(seg string) string {
	if strings.Trim(seg, "/") == "" {
		return ""
	}
	return "/" + strings.Trim(seg, "/")
}

func isExplicitGoAPIHost() (string, bool) {
	h := viper.GetString(KeyGoAPIHost)
	return h, h != ""
}

// GoAPIServiceURL returns a fully assembled go-api service prefix:
//
//	https://{go-api-host}/{pathBase}/gitee-go/{service}
//
// where pathBase is anything between the host and /gitee-go (repo, "sa", ...)
// and service is the service segment (e.g. "/ipipe/rest/v5" or "/sa/rest/v2").
func GoAPIServiceURL(giteeHostname, pathBase, service string) string {
	base := GoAPIBaseURL(giteeHostname, pathBase)
	if strings.Trim(service, "/") == "" {
		return base
	}
	return base + "/" + strings.Trim(service, "/")
}
