## cr push

Push Helm chart packages to an OCI registry

### Synopsis

Push Helm chart packages to an OCI registry.

For every *.tgz file found under --package-path, the command pushes the chart
to <registry-url>/<chart-name>:<chart-version>. If a sibling *.tgz.prov
provenance file is present, it is pushed automatically.

```
cr push [flags]
```

### Options

```
      --ca-file string             Verify registry certificate using this CA bundle
      --cert-file string           Identify registry client using this TLS certificate file
  -h, --help                       help for push
      --insecure-skip-tls-verify   Skip TLS certificate verification
      --key-file string            Identify registry client using this TLS key file
  -p, --package-path string        Path to directory with chart packages (default ".cr-release-packages")
      --password string            Registry password (falls back to ~/.docker/config.json if unset)
      --plain-http                 Use insecure HTTP connections
  -r, --registry-url string        OCI registry URL (e.g. oci://ghcr.io/myorg/charts)
      --skip-existing              Skip pushing chart versions that already exist in the registry
  -u, --username string            Registry username (falls back to ~/.docker/config.json if unset)
```

### Options inherited from parent commands

```
      --config string   Config file (default is $HOME/.cr.yaml)
```

### SEE ALSO

* [cr](cr.md)	 - Helm Chart Repos on Github Pages

