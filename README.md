# Chart Releaser

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![CircleCI](https://circleci.com/gh/helm/chart-releaser/tree/master.svg?style=svg)](https://circleci.com/gh/helm/chart-releaser/tree/master)

**Helps Turn GitHub Repositories into Helm Chart Repositories**

`cr` is a tool designed to help GitHub repos self-host their own chart repos by adding Helm chart artifacts to GitHub Releases named for the chart version and then creating an `index.yaml` file for those releases that can be hosted on GitHub Pages (or elsewhere!).

## Installation

### Binaries (recommended)

Download your preferred asset from the [releases page](https://github.com/helm/chart-releaser/releases) and install manually.

### Go get (for contributing)

```console
$ # clone repo to some directory outside GOPATH
$ git clone github.com/helm/chart-releaser
$ go mod download
$ go install
```

## Usage

Currently, `cr` can create GitHub Releases from a set of charts packaged up into a directory and create an `index.yaml` file for the chart repository from GitHub Releases.

```console
$ cr --help
Create Helm chart repositories on GitHub Pages by uploading Chart packages
and Chart metadata to GitHub Releases and creating a suitable index file

Usage:
  cr [command]

Available Commands:
  help        Help about any command
  index       Update Helm repo index.yaml for the given GitHub repo
  upload      Upload Helm chart packages to GitHub Releases
  version     Print version information

Flags:
      --config string   Config file (default is $HOME/.chart-releaser.yaml)
  -h, --help            help for cr

Use "cr [command] --help" for more information about a command.
```

### Create GitHub Releases from Helm Chart Packages

Scans a path for Helm chart packages and creates releases in the specified GitHub repo uploading the packages.

```console
$ cr upload --help
Upload Helm chart packages to GitHub Releases

Usage:
  cr upload [flags]

Flags:
  -h, --help                  help for upload
  -o, --owner string          GitHub username or organization
  -p, --package-path string   Path to directory with chart packages (default ".cr-release-packages")
  -r, --repo string           GitHub repository
  -t, --token string          GitHub Auth Token

Global Flags:
      --config string   Config file (default is $HOME/.chart-releaser.yaml)
```

### Create the Repository Index from GitHub Releases

Once uploaded you can create an `index.yaml` file that can be hosted on GitHub Pages (or elsewhere).

```console
$ cr index --help

Update a Helm chart repository index.yaml file based on a the
given GitHub repository's releases.

Usage:
  cr index [flags]

Flags:
  -h, --help                  help for index
  -i, --index-path string     Path to index file (default ".cr-index/index.yaml")
  -o, --owner string          GitHub username or organization
  -p, --package-path string   Path to directory with chart packages (default ".cr-release-packages")
  -r, --repo string           GitHub repository
  -t, --token string          GitHub Auth Token (only needed for private repos)

Global Flags:
      --config string   Config file (default is $HOME/.chart-releaser.yaml)
```

## Configuration

`cr` is a command-line application.
All command-line flags can also be set via environment variables or config file.
Environment variables must be prefixed with `CR_`.
Underscores must be used instead of hyphens.

CLI flags, environment variables, and a config file can be mixed.
The following order of precedence applies:

1. CLI flags
1. Environment variables
1. Config file

### Examples

The following example show various ways of configuring the same thing:

#### CLI

    cr upload --owner myaccount --repo helm-charts --package-path .deploy --token 123456789

#### Environment Variables

    export CR_OWNER=myaccount
    export CR_REPO=helm-charts
    export CR_PACKAGE_PATH=.deploy
    export CR_TOKEN="123456789"

    cr upload

#### Config File

`config.yaml`:

```yaml
owner: myaccount
repo: helm-charts
package-path: .deploy
token: 123456789
```

#### Config Usage

    cr upload --config config.yaml


`cr` supports any format [Viper](https://github.com/spf13/viper) can read, i. e. JSON, TOML, YAML, HCL, and Java properties files.

Notice that if no config file is specified, `cr.yaml` (or any of the supported formats) is loaded from the current directory, `$HOME/.cr`, or `/etc/cr`, in that order, if found.
