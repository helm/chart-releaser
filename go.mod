module github.com/helm/chart-releaser

go 1.14

require (
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.3 // indirect
	github.com/Azure/go-autorest/autorest/to v0.3.0 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.2.0 // indirect
	github.com/Songmu/retry v0.1.0
	github.com/google/go-cmp v0.5.2 // indirect
	github.com/google/go-github/v30 v30.0.0
	github.com/goreleaser/goreleaser v0.129.0
	github.com/jmoiron/sqlx v1.2.1-0.20190826204134-d7d95172beb5 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/magefile/mage v1.10.0
	github.com/mattn/go-colorable v0.1.7 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/onsi/ginkgo v1.13.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/rogpeppe/go-internal v1.6.0 // indirect
	github.com/spf13/cobra v1.0.0
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.6.1
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/tools v0.0.0-20200812195022-5ae4c3c160a0
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f // indirect
	helm.sh/helm/v3 v3.4.0-rc.1
	honnef.co/go/tools v0.0.1-2020.1.5 // indirect
	rsc.io/letsencrypt v0.0.3 // indirect; indirect)
)

exclude (
	github.com/Azure/go-autorest v0.9.0
	github.com/Azure/go-autorest v12.0.0+incompatible
)
