package upload

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/ghodss/yaml"
	gh "github.com/google/go-github/github"
	"github.com/paulczar/charthub/pkg/github"
	"github.com/pkg/errors"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/proto/hapi/chart"
)

// Options to be passed in from cli
type Options struct {
	Owner     string
	Repo      string
	Path      string
	Token     string
	Recursive bool
}

type chartPackage struct {
	file  string
	chart *chart.Metadata
}

// ErrReleaseNotFound contains the error for when a release is not found
var (
	ErrReleaseNotFound = errors.New("release is not found")
)

// Packages finds and uploads helm chart packages to github
func Packages(config *Options) error {
	var chartPackages = []chartPackage{}
	var ghc github.GitHub
	var err error
	var ctx = context.TODO()

	//var gitHubClient

	// Create a GitHub client
	ghc, err = github.NewGitHubClient(config.Owner, config.Repo, config.Token)
	if err != nil {
		fmt.Println("failed to log into github")
		os.Exit(1)
	}

	repo, err := ghc.GetRepository(ctx)
	if repo == nil {
		if err != nil {
			panic(err)
		}
		fmt.Printf("Could not find repo %s/%s\n", config.Owner, config.Repo)
		os.Exit(1)
	}
	packages, err := getListOfPackages(config.Path, config.Recursive)
	if err != nil {
		return err
	}
	if len(packages) == 0 {
		fmt.Printf("No charts found at %s, try --recursive or a different path.\n", config.Path)
		os.Exit(1)
	}
	for _, p := range packages {
		m := extractChartMetadataFromPackage(p)
		if m != nil {
			cp := chartPackage{
				file:  p,
				chart: m,
			}
			chartPackages = append(chartPackages, cp)
		}
	}
	for _, pkg := range chartPackages {
		fmt.Printf("--> Processing package %s\n", path.Base(pkg.file))
		req := &gh.RepositoryRelease{
			Name:    &pkg.chart.Name,
			Body:    &pkg.chart.Description,
			TagName: &pkg.chart.Version,
		}
		fmt.Printf("release %#v", *req.TagName)
		release, err := ghc.GetRelease(ctx, *req.TagName)
		if err != nil {
			if err != ErrReleaseNotFound {
				return errors.Wrap(err, "failed to get release")
			}
			fmt.Printf("====> Creating release %s\n", *req.TagName)
			release, err = ghc.CreateRelease(ctx, req)
			if err != nil {
				return err
			}
		} else {
			fmt.Printf("====> Release %s already exists\n", *release.TagName)
		}
		//fmt.Printf("package %s is for chart %s version %s\n", pkg.file, *release.Name, *release.TagName)
		var hasMetadata, hasPackage = false, false
		for _, f := range release.Assets {
			if *f.Name == path.Base(pkg.file) {
				hasPackage = true
				continue
			}
			if *f.Name == "Chart.yaml" {
				hasMetadata = true
				continue
			}
		}
		if hasPackage {
			fmt.Printf("====> Release %s already contains package %s\n", *release.TagName, path.Base(pkg.file))
		} else {
			fmt.Printf("====> Uploading package %s to release %s\n", path.Base(pkg.file), *release.TagName)
			_, err := ghc.UploadAsset(ctx, *release.ID, pkg.file)
			if err != nil {
				return errors.Wrapf(err,
					"failed to upload asset: %s", pkg.file)
			}
		}
		if hasMetadata {
			fmt.Printf("====> Release %s already contains Chart.yaml\n", *release.TagName)
		} else {
			fmt.Printf("====> Uploading Chart.yaml to release %s\n", *release.TagName)
			dir, err := ioutil.TempDir("", *release.Name)
			if err != nil {
				log.Fatal(err)
			}
			defer os.RemoveAll(dir)
			f := path.Join(dir, "Chart.yaml")
			b, err := yaml.Marshal(pkg.chart)
			if err != nil {
				return err
			}
			if err = ioutil.WriteFile(f, b, 0644); err != nil {
				return err
			}
			_, err = ghc.UploadAsset(ctx, *release.ID, f)
			if err != nil {
				return errors.Wrapf(err,
					"failed to upload asset: %s", f)
			}
		}
	}

	return nil
}

func getListOfPackages(dir string, recurse bool) ([]string, error) {
	archives, err := filepath.Glob(filepath.Join(dir, "*.tgz"))
	if err != nil {
		return nil, err
	}
	if recurse {
		moreArchives, err := filepath.Glob(filepath.Join(dir, "**/*.tgz"))
		if err != nil {
			return nil, err
		}
		archives = append(archives, moreArchives...)
	}
	return archives, nil
}

func extractChartMetadataFromPackage(pkg string) *chart.Metadata {
	c, err := chartutil.Load(pkg)
	if err != nil {
		return nil
	}
	return c.Metadata
}
