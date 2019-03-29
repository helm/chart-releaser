# Chart Releaser turns github repositories into Helm Chart Repositories

Chart Releaser is a tool designed to help github repos self-host their own chart repos by adding Helm Chart artifacts to github releases named for the chart version and then creating an `index.yaml` find for those releases that can be hosted in github pages (or elsewhere!).

The examples below were used to create the `demo` chart found at [paulczar/helm-demo](https://github.com/paulczar/helm-demo) which has github pages configured to serve the `docs` directory of the `master` branch.

## Usage

```console
$ go install github.com/helm/chart-releaser
$ chart-releaser
chart-releaser creates helm chart repositories on github pages by uploading Chart packages
and Chart metadata to github releases and creating a suitable index file.

Usage:
  chart-releaser [command]

Available Commands:
  help        Help about any command
  index       creates helm repo index.yaml for given github repo
  upload      Uploads Helm Chart packages to github releases

Flags:
      --config string   config file (default is $HOME/.chart-releaser.yaml)
  -h, --help            help for chart-releaser
  -t, --toggle          Help message for toggle

Use "chart-releaser [command] --help" for more information about a command.
```

### Upload Helm Chart Packages

Scans a path for Helm Chart Package and creates a release in the specified github repo then uploads the package and Chart.yaml

```console
$ chart-releaser upload -o paulczar -r helm-demo -t $TOKEN -p ~/development/scratch/helm/demo/ --recursive
--> Processing package demo-0.1.0.tgz
release "0.1.0"====> Release 0.1.0 already exists
====> Release 0.1.0 already contains package demo-0.1.0.tgz
====> Release 0.1.0 already contains Chart.yaml
--> Processing package demo-0.1.10.tgz
release "0.1.10"====> Release 0.1.10 already exists
====> Release 0.1.10 already contains package demo-0.1.10.tgz
====> Release 0.1.10 already contains Chart.yaml
...
```

### Create Index from Github releases

Once uploaded you can create an `index.yaml` file that can be hosted in github pages (or elsewhere).

```console
$ chart-releaser index -o paulczar -r helm-demo -t $TOKEN -p ~/development/scratch/helm/demo/docs/index.yaml
====> Using existing index at /home/pczarkowski/development/scratch/helm/demo/docs/index.yaml
--> Checking for releases with helm chart packages
====> Found demo-0.1.0.tgz
====> Found demo-0.1.14.tgz
...
--> Updating index /home/pczarkowski/development/scratch/helm/demo/docs/index.yaml
```

### Using the resultant Helm Chart Repository

```console
$ helm repo add demo https://tech.paulcz.net/helm-demo
"demo" has been added to your repositories
$ helm install --repo http://tech.paulcz.net/helm-demo demo
NAME:   kindred-stoat
...
```

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
