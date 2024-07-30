# Chart Releaser

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
![CI](https://github.com/helm/chart-releaser/workflows/CI/badge.svg?branch=main&event=push)

**Helps Turn GitHub Repositories into Helm Chart Repositories**

`cr` is a tool designed to help GitHub repos self-host their own chart repos by adding Helm chart artifacts to GitHub Releases named for the chart version and then creating an `index.yaml` file for those releases that can be hosted on GitHub Pages (or elsewhere!).

## Installation

### Binaries (recommended)

Download your preferred asset from the [releases page](https://github.com/helm/chart-releaser/releases) and install manually.

### Homebrew

```console
$ brew tap helm/tap
$ brew install chart-releaser
```

### Go get (for contributing)

```console
// clone repo to some directory outside GOPATH

$ git clone https://github.com/helm/chart-releaser
$ cd chart-releaser
$ go mod download
$ go install ./...
```

### Docker (for Continuous Integration)

Docker images are pushed to the [helmpack/chart-releaser](https://quay.io/repository/helmpack/chart-releaser?tab=tags) Quay container registry. The Docker image is built on top of Alpine and its default entry-point is `cr`. See the [Dockerfile](./Dockerfile) for more details.

## Common Usage

Currently, `cr` can create GitHub Releases from a set of charts packaged up into a directory and create an `index.yaml` file for the chart repository from GitHub Releases.

```console
$ cr --help
Create Helm chart repositories on GitHub Pages by uploading Chart packages
and Chart metadata to GitHub Releases and creating a suitable index file

Usage:
  cr [command]

Available Commands:
  completion  generate the autocompletion script for the specified shell
  help        Help about any command
  index       Update Helm repo index.yaml for the given GitHub repo
  package     Package Helm charts
  upload      Upload Helm chart packages to GitHub Releases
  version     Print version information

Flags:
      --config string   Config file (default is $HOME/.cr.yaml)
  -h, --help            help for cr

Use "cr [command] --help" for more information about a command.
```

### Dealing with charts that have dependencies

Unfortuntely the releaser-tool won't automatically add repositories for dependencies, and this needs to be added to your pipeline ([example](https://github.com/davidkarlsen/flyway-operator/blob/main/.github/workflows/chart-release.yaml#L31)), prior to running the releaser, like this:

```yaml
    - name: add repos
        run: |
          helm repo add bitnami https://charts.bitnami.com/bitnami
          helm repo add bitnami-pre2022 https://raw.githubusercontent.com/bitnami/charts/eb5f9a9513d987b519f0ecd732e7031241c50328/bitnami

      - name: Run chart-releaser
        uses: helm/chart-releaser-action@v1.6.0
        with:
          charts_dir: config/helm-chart
        env:
          CR_TOKEN: "${{ secrets.GITHUB_TOKEN }}"
```


### Create GitHub Releases from Helm Chart Packages

Scans a path for Helm chart packages and creates releases in the specified GitHub repo uploading the packages.

```console
$ cr upload --help
Upload Helm chart packages to GitHub Releases

Usage:
  cr upload [flags]

Flags:
  -c, --commit string                  Target commit for release
      --generate-release-notes         Whether to automatically generate the name and body for this release. See https://docs.github.com/en/rest/releases/releases
  -b, --git-base-url string            GitHub Base URL (only needed for private GitHub) (default "https://api.github.com/")
  -r, --git-repo string                GitHub repository
  -u, --git-upload-url string          GitHub Upload URL (only needed for private GitHub) (default "https://uploads.github.com/")
  -h, --help                           help for upload
  -o, --owner string                   GitHub username or organization
  -p, --package-path string            Path to directory with chart packages (default ".cr-release-packages")
      --release-name-template string   Go template for computing release names, using chart metadata (default "{{ .Name }}-{{ .Version }}")
      --release-notes-file string      Markdown file with chart release notes. If it is set to empty string, or the file is not found, the chart description will be used instead. The file is read from the chart package
      --skip-existing                  Skip upload if release exists
  -t, --token string                   GitHub Auth Token
      --make-release-latest bool       Mark the created GitHub release as 'latest' (default "true")
      --packages-with-index            Host the package files in the GitHub Pages branch

Global Flags:
      --config string   Config file (default is $HOME/.cr.yaml)
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
  -b, --git-base-url string            GitHub Base URL (only needed for private GitHub) (default "https://api.github.com/")
  -r, --git-repo string                GitHub repository
  -u, --git-upload-url string          GitHub Upload URL (only needed for private GitHub) (default "https://uploads.github.com/")
  -h, --help                           help for index
  -i, --index-path string              Path to index file (default ".cr-index/index.yaml")
  -o, --owner string                   GitHub username or organization
  -p, --package-path string            Path to directory with chart packages (default ".cr-release-packages")
      --pages-branch string            The GitHub pages branch (default "gh-pages")
      --pages-index-path string        The GitHub pages index path (default "index.yaml")
      --pr                             Create a pull request for index.yaml against the GitHub Pages branch (must not be set if --push is set)
      --push                           Push index.yaml to the GitHub Pages branch (must not be set if --pr is set)
      --release-name-template string   Go template for computing release names, using chart metadata (default "{{ .Name }}-{{ .Version }}")
      --remote string                  The Git remote used when creating a local worktree for the GitHub Pages branch (default "origin")
  -t, --token string                   GitHub Auth Token (only needed for private repos)
      --packages-with-index            Host the package files in the GitHub Pages branch

Global Flags:
      --config string   Config file (default is $HOME/.cr.yaml)
```

## Usage with a private repository

When using this tool on a private repository, helm is unable to download the chart package files. When you give Helm your username and password it uses it to authenticate to the repository (the index file). The index file then tells Helm where to get the tarball. If the tarball is hosted in some other location (Github Releases in this case) then it would require a second authentication (which Helm does not support). The solution is to host the files in the same place as your index file and make the links relative paths so there is no need for the second authentication.

[#123](https://github.com/helm/chart-releaser/pull/123) solve this by adding a `--packages-with-index` flag to the upload and index commands.

### Prerequisites

Have a Github token with the right permissions (SSO enabled for entreprise) and Github Pages configured.

### Usage

Here are the three commands you must run for a chart to end-up hosted in the root directory of your Github page and be accessible :

```bash
cr package <chart>
```

```bash
cr upload --owner <owner> --git-repo <repo_name> --packages-with-index --token <token> --push --skip-existing
```

Don't forget the `--skip-existing` flag in the upload command to avoid getting a `422 Validation Failed` error.

```bash
cr index --owner <owner> --git-repo <repo_name>  --packages-with-index --index-path . --token <token> --push
```

### Example

With a **testChart** helm chart in the root of your repository :

```bash
cr package testChart/
```

You will obtain the .tgz in the **./cr-release-pacakges**

![repository](https://user-images.githubusercontent.com/116822264/227932604-50bf08d9-44bc-4170-b830-a2dccf1afdde.PNG)

Do the two followng commands :

```bash
cr upload --owner <owner> --git-repo <repo_name> --packages-with-index --token <token> --push --skip-existing

cr index --owner <owner> --git-repo <repo_name>  --packages-with-index --index-path . --token <token> --push
```

You should obtain a release of your **chart** as well as the .tgz in the root of your **github-pages** branch.

![github pages](https://user-images.githubusercontent.com/116822264/230021850-8d713d5d-426b-41ac-9ff4-c7e19de40e32.PNG)

With the `index.yaml` that references each chart and every different versions of those charts :

![index](https://user-images.githubusercontent.com/116822264/230022176-ec57a8e7-ddb7-4318-a03d-032548421e21.PNG)

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

```bash
cr upload --owner myaccount --git-repo helm-charts --package-path .deploy --token 123456789
```

#### Environment Variables

```bash
export CR_OWNER=myaccount
export CR_GIT_REPO=helm-charts
export CR_PACKAGE_PATH=.deploy
export CR_TOKEN="123456789"
export CR_GIT_BASE_URL="https://api.github.com/"
export CR_GIT_UPLOAD_URL="https://uploads.github.com/"
export CR_SKIP_EXISTING=true

cr upload
```

#### Config File

`config.yaml`:

```yaml
owner: myaccount
git-repo: helm-charts
package-path: .deploy
token: 123456789
git-base-url: https://api.github.com/
git-upload-url: https://uploads.github.com/
```

#### Config Usage

```bash
cr upload --config config.yaml
```

`cr` supports any format [Viper](https://github.com/spf13/viper) can read, i.e. JSON, TOML, YAML, HCL, and Java properties files.

Notice that if no config file is specified, `cr.yaml` (or any of the supported formats) is loaded from the current directory, `$HOME/.cr`, or `/etc/cr`, in that order, if found.

#### Notes for Github Enterprise Users

For Github Enterprise, `chart-releaser` users need to set `git-base-url` and `git-upload-url` correctly, but the correct values are not always obvious to endusers.

By default they are often along these lines:

```console
https://ghe.example.com/api/v3/
https://ghe.example.com/api/uploads/
```

If you are trying to figure out what your `upload_url` is try to use a curl command like this:
`curl -u username:token https://example.com/api/v3/repos/org/repo/releases`
and then look for `upload_url`. You need the part of the URL that appears before `repos/` in the path.

## Common Error Messages

During the upload, you can get the following error :

```bash
422 Validation Failed [{Resource:Release Field:tag_name Code:already_exists Message:}]
```

You can solve it by adding the `--skip-existing` flag to your command. More details can be found in [#101](https://github.com/helm/chart-releaser/issues/101#issuecomment-766410614) and in [#111](https://github.com/helm/chart-releaser/pull/111) that solved this.

## Known Bug

Currently, if you set the upload URL incorrectly, let's say to something like `https://example.com/uploads/`, then `cr upload` will appear to work, but the release will not be complete. When everything is working there should be three assets in each release, but instead, there will only be two source code assets. The third asset is missing and is needed by Helm. This issue will become apparent when you run `cr index` and it always claims that nothing has changed, because it can't find the asset it expects for the release.

It appears like the [go-github Do call](https://github.com/google/go-github/blob/master/github/github.go#L520) does not catch the fact that the upload URL is incorrect and passes back the expected error. If the asset upload fails, it would be better if the release was rolled back (deleted) and an appropriate log message is displayed to the user.

The `cr index` command should also generate a warning when a release has no assets attached to it, to help people detect and troubleshoot this type of problem.
