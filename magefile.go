// Copyright The Helm Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build mage

package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

func Lint() error {
	if err := sh.RunV("bash", "-c", "shopt -s globstar; shellcheck **/*.sh"); err != nil {
		return err
	}
	if err := sh.RunV("golangci-lint", "run", "--timeout", "3m"); err != nil {
		return err
	}
	if err := sh.RunV("go", "vet", "-v", "./..."); err != nil {
		return err
	}
	if err := sh.RunV("goimports", "-w", "-l", "."); err != nil {
		return err
	}
	return sh.RunV("git", "diff", "--exit-code")
}

func CheckLicenseHeaders() error {
	var checkFailed bool

	if err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext == ".sh" || ext == ".go" {
			fmt.Print("Checking ", path, " ")

			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()

			var hasCopyright bool
			var hasLicense bool

			scanner := bufio.NewScanner(f)
			// only check first 20 lines
			for i := 0; i < 20 && scanner.Scan(); i++ {
				line := scanner.Text()
				if !hasCopyright && strings.Contains(line, "Copyright The Helm Authors") {
					hasCopyright = true
				}
				if !hasLicense && strings.Contains(line, "https://www.apache.org/licenses/LICENSE-2.0") {
					hasLicense = true
				}
			}

			if !(hasCopyright && hasLicense) {
				fmt.Println("❌")
				checkFailed = true
			} else {
				fmt.Println("☑️")
			}

			return nil
		}
		return nil
	}); err != nil {
		return err
	}

	if checkFailed {
		return errors.New("file(s) without license header found")
	}
	return nil
}

func Test() error {
	return sh.RunV("go", "test", "./...", "-race")
}

func Build() error {
	return sh.RunV("goreleaser", "release", "--rm-dist", "--snapshot")
}

func Release() error {
	mg.Deps(Test)
	return sh.RunV("goreleaser", "release", "--rm-dist")
}
