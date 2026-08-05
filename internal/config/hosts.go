package config

import (
	"errors"
	"fmt"
	"os"

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
