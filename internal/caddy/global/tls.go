package global

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/caddyserver/ingress/pkg/converter"
	"github.com/caddyserver/ingress/pkg/store"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TLSPlugin struct {
	secretsDir     string
	secretVersions map[string]string
}

func (p *TLSPlugin) IngressPlugin() converter.PluginInfo {
	return converter.PluginInfo{
		Name: "tls",
		New:  func() converter.Plugin { return new(TLSPlugin) },
	}
}

func init() {
	converter.RegisterPlugin(&TLSPlugin{})
}

func (p *TLSPlugin) GlobalHandler(config *converter.Config, store *store.Store) error {
	tlsApp := config.GetTLSApp()
	httpServer := config.GetHTTPServer()
	if p.secretVersions == nil {
		p.secretVersions = make(map[string]string)
	}

	var hosts []string
	var secretNames []string

	// Get all Hosts and SecretNames subject to custom TLS certs
	for _, ing := range store.Ingresses() {
		for _, tlsRule := range ing.Spec.TLS {
			for _, h := range tlsRule.Hosts {
				if !slices.Contains(hosts, h) {
					hosts = append(hosts, h)
				}

				s := fmt.Sprintf("%s/%s", ing.Namespace, tlsRule.SecretName)
				if !slices.Contains(secretNames, s) {
					secretNames = append(secretNames, s)
				}
			}
		}
	}

	// Evict secrets that are no longer needed, or outdated per our informer.
	for name, version := range p.secretVersions {
		keep := true
		if !slices.Contains(secretNames, name) {
			store.Logger.Infof("TLS secret dereferenced (%s)", name)
			keep = false
		} else {
			currentMeta := store.SecretMeta(name)
			if currentMeta != nil && version != currentMeta.ResourceVersion {
				keep = false
			}
		}

		if !keep {
			delete(p.secretVersions, name)
			if path, err := p.secretPath(name); err != nil {
				return err
			} else if err := os.Remove(path); err != nil {
				return err
			}
		}
	}

	// Fetch missing secrets.
	// Note: The plugin is called many more times than secrets typically update.
	// This code should thus try to keep disk accesses to a minimum.
	// Note: The nil check here is to accomodate tests.
	if store.KubeClient != nil {
		for _, name := range secretNames {
			if _, ok := p.secretVersions[name]; !ok {
				store.Logger.Infof("TLS secret updated (%s)", name)

				parts := strings.SplitN(name, "/", 2)
				secret, err := store.KubeClient.CoreV1().Secrets(parts[0]).Get(context.TODO(), parts[1], v1.GetOptions{})
				if err != nil {
					return err
				}

				p.secretVersions[name] = secret.ResourceVersion

				content := make([]byte, 0)
				for _, cert := range secret.Data {
					content = append(content, cert...)
				}
				if path, err := p.secretPath(name); err != nil {
					return err
				} else if err := os.WriteFile(path, content, 0600); err != nil {
					return err
				}

				// TODO: Can we secure erase secret data from memory?
			}
		}
	}

	if len(hosts) > 0 {
		// TODO: This does not detect changes to the secrets themselves.
		tlsApp.CertificatesRaw["load_folders"] = json.RawMessage(`["` + p.secretsDir + `"]`)
		// do not manage certificates for those hosts
		httpServer.AutoHTTPS.SkipCerts = hosts
	}
	return nil
}

// secretPath builds a file path from a secret name
func (p *TLSPlugin) secretPath(secretName string) (string, error) {
	if p.secretsDir == "" {
		dir, err := os.MkdirTemp("", "caddy-ingress-")
		if err != nil {
			return "", err
		}
		p.secretsDir = dir
	}

	var s string
	for _, c := range secretName {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '-' || c == '_' {
			s += string(c)
		} else {
			s += "__"
		}
	}
	return filepath.Join(p.secretsDir, s+".pem"), nil
}

// Interface guards
var (
	_ = converter.GlobalMiddleware(&TLSPlugin{})
)
