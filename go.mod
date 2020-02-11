module github.com/helm/chart-releaser

go 1.12

require (
	github.com/Songmu/retry v0.0.1
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/pkg/errors v0.8.1
	github.com/spf13/cobra v0.0.5
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.3.2
	github.com/stretchr/testify v1.4.0
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	helm.sh/helm/v3 v3.0.3
)

// Transitive requirement from Helm.
replace github.com/docker/docker => github.com/docker/docker v0.7.3-0.20190327010347-be7ac8be2ae0
