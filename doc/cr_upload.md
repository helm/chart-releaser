## cr upload

Upload Helm chart packages to GitHub Releases

### Synopsis

Upload Helm chart packages to GitHub Releases

```
cr upload [flags]
```

### Options

```
  -c, --commit string                  Target commit for release
      --generate-release-notes         Whether to automatically generate the name and body for this release. See https://docs.github.com/en/rest/releases/releases
  -b, --git-base-url string            GitHub Base URL (only needed for private GitHub) (default "https://api.github.com/")
  -r, --git-repo string                GitHub repository
  -u, --git-upload-url string          GitHub Upload URL (only needed for private GitHub) (default "https://uploads.github.com/")
  -h, --help                           help for upload
      --make-release-latest            Mark the created GitHub release as 'latest' (default true)
  -o, --owner string                   GitHub username or organization
  -p, --package-path string            Path to directory with chart packages (default ".cr-release-packages")
      --packages-with-index            Host the package files in the GitHub Pages branch
      --pages-branch string            The GitHub pages branch (default "gh-pages")
      --pr                             Create a pull request for the chart package against the GitHub Pages branch (must not be set if --push is set)
      --prerelease                     Mark this as 'Pre-release' (default: false)
      --push                           Push the chart package to the GitHub Pages branch (must not be set if --pr is set)
      --release-name-template string   Go template for computing release names, using chart metadata (default "{{ .Name }}-{{ .Version }}")
      --release-notes-file string      Markdown file with chart release notes. If it is set to empty string, or the file is not found, the chart description will be used instead. The file is read from the chart package
      --remote string                  The Git remote used when creating a local worktree for the GitHub Pages branch (default "origin")
      --skip-existing                  Skip upload if release exists
  -t, --token string                   GitHub Auth Token
```

### Options inherited from parent commands

```
      --config string   Config file (default is $HOME/.cr.yaml)
```

### SEE ALSO

* [cr](cr.md)	 - Helm Chart Repos on Github Pages

