module github.com/helm/chart-releaser

go 1.14

require (
	github.com/Azure/go-autorest/autorest/azure/auth v0.4.2 // indirect
	github.com/Azure/go-autorest/autorest/to v0.3.0 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.2.0 // indirect
	github.com/Songmu/retry v0.1.0
	github.com/google/go-github/v30 v30.0.0
	github.com/goreleaser/goreleaser v0.129.0
	github.com/magefile/mage v1.10.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/onsi/ginkgo v1.11.0 // indirect
	github.com/onsi/gomega v1.8.1 // indirect
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v0.0.6
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.6.2
	github.com/stretchr/testify v1.5.1
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/tools v0.0.0-20200204192400-7124308813f3
	helm.sh/helm/v3 v3.1.2
	rsc.io/letsencrypt v0.0.3 // indirect
)

exclude (
	github.com/Azure/go-autorest v0.9.0
	github.com/Azure/go-autorest v12.0.0+incompatible
)
