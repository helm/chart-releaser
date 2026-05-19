// Copyright The Helm Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pusher

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/registry"

	"github.com/helm/chart-releaser/pkg/config"
)

const ociScheme = "oci://"

// Pusher pushes Helm chart packages to an OCI registry.
type Pusher struct {
	config *config.Options
	out    io.Writer
}

// NewPusher returns a configured Pusher.
func NewPusher(cfg *config.Options) *Pusher {
	return &Pusher{config: cfg, out: os.Stdout}
}

// PushPackages pushes every *.tgz package found under PackagePath to the
// configured OCI registry. Sibling *.tgz.prov files are picked up
// automatically by Helm's push action.
func (p *Pusher) PushPackages() error {
	if p.config.RegistryURL == "" {
		return errors.New("'--registry-url' is required")
	}
	if !strings.HasPrefix(p.config.RegistryURL, ociScheme) {
		return fmt.Errorf("registry-url must start with %q (got %q)", ociScheme, p.config.RegistryURL)
	}

	chartPackages, err := filepath.Glob(filepath.Join(p.config.PackagePath, "*.tgz"))
	if err != nil {
		return err
	}
	if len(chartPackages) == 0 {
		return fmt.Errorf("no chart packages found in %s", p.config.PackagePath)
	}

	registryClient, err := newRegistryClient(p.config)
	if err != nil {
		return fmt.Errorf("creating registry client: %w", err)
	}

	settings := cli.New()
	cfg := &action.Configuration{RegistryClient: registryClient}

	pushClient := action.NewPushWithOpts(
		action.WithPushConfig(cfg),
		action.WithTLSClientConfig(p.config.CertFile, p.config.KeyFile, p.config.CAFile),
		action.WithInsecureSkipTLSVerify(p.config.InsecureSkipTLSVerify),
		action.WithPlainHTTP(p.config.PlainHTTP),
		action.WithPushOptWriter(p.out),
	)
	pushClient.Settings = settings

	remote := strings.TrimRight(p.config.RegistryURL, "/")

	for _, chartPackage := range chartPackages {
		if p.config.SkipExisting {
			exists, err := chartExists(registryClient, remote, chartPackage)
			if err != nil {
				return fmt.Errorf("checking %s in registry: %w", chartPackage, err)
			}
			if exists {
				fmt.Fprintf(p.out, "Skipping %s: already exists in %s\n", chartPackage, remote)
				continue
			}
		}

		output, err := pushClient.Run(chartPackage, remote)
		if err != nil {
			return fmt.Errorf("pushing %s to %s: %w", chartPackage, remote, err)
		}
		if output != "" {
			fmt.Fprint(p.out, output)
		}
		fmt.Fprintf(p.out, "Successfully pushed %s to %s\n", chartPackage, remote)
	}
	return nil
}

// chartExists reports whether the chart in the given .tgz is already present
// in the registry at remote. Helm replaces '+' with '_' in version tags;
// chartExists applies the same substitution before resolving.
func chartExists(client *registry.Client, remote, chartPackage string) (bool, error) {
	ch, err := loader.LoadFile(chartPackage)
	if err != nil {
		return false, err
	}
	ref := fmt.Sprintf("%s/%s:%s",
		strings.TrimPrefix(remote, ociScheme),
		ch.Metadata.Name,
		strings.ReplaceAll(ch.Metadata.Version, "+", "_"),
	)
	if _, err := client.Resolve(ref); err != nil {
		return false, nil
	}
	return true, nil
}

func newRegistryClient(cfg *config.Options) (*registry.Client, error) {
	opts := []registry.ClientOption{
		registry.ClientOptEnableCache(true),
		registry.ClientOptWriter(io.Discard),
		registry.ClientOptBasicAuth(cfg.Username, cfg.Password),
	}

	useTLS := cfg.CertFile != "" || cfg.KeyFile != "" || cfg.CAFile != "" || cfg.InsecureSkipTLSVerify
	if useTLS {
		tlsConf, err := newClientTLS(cfg.CertFile, cfg.KeyFile, cfg.CAFile, cfg.InsecureSkipTLSVerify)
		if err != nil {
			return nil, fmt.Errorf("building TLS config: %w", err)
		}
		opts = append(opts, registry.ClientOptHTTPClient(&http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsConf,
				Proxy:           http.ProxyFromEnvironment,
			},
		}))
	}
	if cfg.PlainHTTP {
		opts = append(opts, registry.ClientOptPlainHTTP())
	}

	return registry.NewClient(opts...)
}

func newClientTLS(certFile, keyFile, caFile string, insecureSkipTLSVerify bool) (*tls.Config, error) {
	c := &tls.Config{InsecureSkipVerify: insecureSkipTLSVerify} // #nosec G402 -- gated by --insecure-skip-tls-verify flag

	if certFile != "" && keyFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("loading key pair from %s and %s: %w", certFile, keyFile, err)
		}
		c.Certificates = []tls.Certificate{cert}
	}
	if caFile != "" {
		b, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("reading CA file %s: %w", caFile, err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(b) {
			return nil, fmt.Errorf("no certificates found in CA file %s", caFile)
		}
		c.RootCAs = pool
	}
	return c, nil
}
