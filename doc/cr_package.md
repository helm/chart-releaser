## cr package

Package Helm charts

### Synopsis

This command packages a chart into a versioned chart archive file. If a path
is given, this will look at that path for a chart (which must contain a
Chart.yaml file) and then package that directory.


If you wish to use advanced packaging options such as creating signed
packages or updating chart dependencies please use "helm package" instead.

```
cr package [CHART_PATH] [...] [flags]
```

### Options

```
  -h, --help                     help for package
      --key string               Name of the key to use when signing
      --keyring string           Location of a public keyring (default "~/.gnupg/pubring.gpg")
  -p, --package-path string      Path to directory with chart packages (default ".cr-release-packages")
      --passphrase-file string   Location of a file which contains the passphrase for the signing key. Use '-' in order to read from stdin
      --sign                     Use a PGP private key to sign this package
```

### Options inherited from parent commands

```
      --config string   Config file (default is $HOME/.cr.yaml)
```

### SEE ALSO

* [cr](cr.md)	 - Helm Chart Repos on Github Pages

