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
```

### Options inherited from parent commands

```
      --config string   Config file (default is $HOME/.cr.yaml)
```

### SEE ALSO

* [cr](cr.md)	 - Helm Chart Repos on Github Pages

