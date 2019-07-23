// Copyright The Helm Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"fmt"
	"github.com/mitchellh/go-homedir"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	homeDir, _            = homedir.Dir()
	configSearchLocations = []string{
		".",
		filepath.Join(homeDir, ".cr"),
		"/etc/cr",
	}
)

type Options struct {
	Owner       string `mapstructure:"owner"`
	GitRepo     string `mapstructure:"git-repo"`
	ChartsRepo  string `mapstructure:"charts-repo"`
	IndexPath   string `mapstructure:"index-path"`
	PackagePath string `mapstructure:"package-path"`
	Token       string `mapstructure:"token"`
}

func LoadConfiguration(cfgFile string, cmd *cobra.Command, requiredFlags []string) (*Options, error) {
	v := viper.New()

	cmd.Flags().VisitAll(func(flag *flag.Flag) {
		flagName := flag.Name
		if flagName != "config" && flagName != "help" {
			if err := v.BindPFlag(flagName, flag); err != nil {
				// can't really happen
				panic(fmt.Sprintln(errors.Wrapf(err, "Error binding flag '%s'", flagName)))
			}
		}
	})

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.SetEnvPrefix("CR")

	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		v.SetConfigName("cr")
		for _, searchLocation := range configSearchLocations {
			v.AddConfigPath(searchLocation)
		}
	}

	if err := v.ReadInConfig(); err != nil {
		if cfgFile != "" {
			// Only error out for specified config file. Ignore for default locations.
			return nil, errors.Wrap(err, "Error loading config file")
		}
	} else {
		fmt.Println("Using config file: ", v.ConfigFileUsed())
	}

	opts := &Options{}
	if err := v.Unmarshal(opts); err != nil {
		return nil, errors.Wrap(err, "Error unmarshaling configuration")
	}

	elem := reflect.ValueOf(opts).Elem()
	for _, requiredFlag := range requiredFlags {
		fieldName := kebabCaseToTitleCamelCase(requiredFlag)
		f := elem.FieldByName(fieldName)
		value := fmt.Sprintf("%v", f.Interface())
		if value == "" {
			return nil, errors.Errorf("'--%s' is required", requiredFlag)
		}
	}

	return opts, nil
}

func kebabCaseToTitleCamelCase(input string) (result string) {
	nextToUpper := true
	for _, runeValue := range input {
		if nextToUpper {
			result += strings.ToUpper(string(runeValue))
			nextToUpper = false
		} else {
			if runeValue == '-' {
				nextToUpper = true
			} else {
				result += string(runeValue)
			}
		}
	}
	return
}
