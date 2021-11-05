module github.com/helm/chart-releaser

go 1.16

require (
	github.com/Songmu/retry v0.1.0
	github.com/golangci/golangci-lint v1.42.1
	github.com/google/go-github/v36 v36.0.0
	github.com/goreleaser/goreleaser v0.183.0
	github.com/magefile/mage v1.11.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.2.1
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.9.0
	github.com/stretchr/testify v1.7.0
	golang.org/x/oauth2 v0.0.0-20211005180243-6b3c2da341f1
	golang.org/x/tools v0.1.7
	helm.sh/helm/v3 v3.7.1
)

replace (
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d
	github.com/docker/docker => github.com/moby/moby v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible
)
