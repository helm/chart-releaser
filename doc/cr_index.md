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
  -b, --git-base-url string            GitHub Base URL (only needed for private GitHub) (default "https://api.github.com/")
  -r, --git-repo string                GitHub repository
  -u, --git-upload-url string          GitHub Upload URL (only needed for private GitHub) (default "https://uploads.github.com/")
  -h, --help                           help for index
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
  -t, --token string                   GitHub Auth Token (only needed for private repos)
```

### Options inherited from parent commands

```
      --config string   Config file (default is $HOME/.cr.yaml)
```

### SEE ALSO

* [cr](cr.md)	 - Helm Chart Repos on Github Pages

