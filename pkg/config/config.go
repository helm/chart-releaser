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
	"os"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Options struct {
	Owner     string
	Repo      string
	Path      string
	Token     string
	Recursive bool
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
	v.SetEnvPrefix("CH")

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".charthub" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".charthub")
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
		f := elem.FieldByName(strings.Title(requiredFlag))
		value := fmt.Sprintf("%v", f.Interface())
		if value == "" {
			return nil, errors.Errorf("'--%s' is required", requiredFlag)
		}
	}

	return opts, nil
}

