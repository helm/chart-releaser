## cr index

Update Helm repo index.yaml for the given GitHub repo

### Synopsis


Update a Helm chart repository index.yaml file based on a the
given GitHub repository's releases.
	

```
cr index [flags]
```

### Options

```
      --exclude strings                Skip chart packages whose file name matches one of these glob patterns, e.g. '*-rc*'; takes precedence over --include (can be repeated or comma-separated)
  -b, --git-base-url string            GitHub Base URL (only needed for private GitHub) (default "https://api.github.com/")
  -r, --git-repo string                GitHub repository
  -u, --git-upload-url string          GitHub Upload URL (only needed for private GitHub) (default "https://uploads.github.com/")
  -h, --help                           help for index
      --include strings                Only process chart packages whose file name matches one of these glob patterns, e.g. 'my-chart-*' (can be repeated or comma-separated)
      --index-commit-message string    Go template for the commit message of the index update, e.g. 'chore: update {{ .PagesIndexPath }}' (default "Update {{ .PagesIndexPath }}")
  -i, --index-path string              Path to index file (default ".cr-index/index.yaml")
  -o, --owner string                   GitHub username or organization
  -p, --package-path string            Path to directory with chart packages (default ".cr-release-packages")
      --packages-with-index            Host the package files in the GitHub Pages branch
      --pages-branch string            The GitHub pages branch (default "gh-pages")
      --pages-index-path string        The GitHub pages index path (default "index.yaml")
      --pr                             Create a pull request for index.yaml against the GitHub Pages branch (must not be set if --push is set)
      --push                           Push index.yaml to the GitHub Pages branch (must not be set if --pr is set)
      --release-name-template string   Go template for computing release names, using chart metadata (default "{{ .Name }}-{{ .Version }}")
      --remote string                  The Git remote used when creating a local worktree for the GitHub Pages branch (default "origin")
      --replace-existing               Update existing chart versions using the supplied packages, including checksums
      --sign-commits                   Create the index commit through GitHub's API for GitHub signing (requires --push and --token; incompatible with --pr)
  -t, --token string                   GitHub Auth Token (only needed for private repos)
      --use-api-asset-url              Use GitHub API asset URLs in the index (requires a client that sends Accept: application/octet-stream)
```

### Options inherited from parent commands

```
      --config string   Config file (default is $HOME/.cr.yaml)
```

### SEE ALSO

* [cr](cr.md)	 - Helm Chart Repos on Github Pages

