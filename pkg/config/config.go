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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/mitchellh/go-homedir"

	"github.com/cpepper96/zarf-testing/pkg/util"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	homeDir, _            = homedir.Dir()
	configSearchLocations = []string{
		".",
		".zt",
		".zarf-testing",
		filepath.Join(homeDir, ".zt"),
		filepath.Join(homeDir, ".zarf-testing"),
		"/usr/local/etc/zt",
		"/etc/zt",
		// Legacy chart-testing paths for migration
		".ct",
		filepath.Join(homeDir, ".ct"),
		"/usr/local/etc/ct",
		"/etc/ct",
	}
)

type Configuration struct {
	// Git-related configuration
	Remote                  string        `mapstructure:"remote"`
	TargetBranch            string        `mapstructure:"target-branch"`
	Since                   string        `mapstructure:"since"`
	
	// General configuration
	BuildID                 string        `mapstructure:"build-id"`
	Debug                   bool          `mapstructure:"debug"`
	GithubGroups            bool          `mapstructure:"github-groups"`
	
	// Zarf package configuration
	ZarfDirs                []string      `mapstructure:"zarf-dirs"`
	ExcludedPackages        []string      `mapstructure:"excluded-packages"`
	Packages                []string      `mapstructure:"packages"`
	ProcessAllPackages      bool          `mapstructure:"all"`
	
	// Validation configuration
	CheckVersionIncrement   bool          `mapstructure:"check-version-increment"`
	ValidateImagePinning    bool          `mapstructure:"validate-image-pinning"`
	ValidatePackageSchema   bool          `mapstructure:"validate-package-schema"`
	ValidateComponents      bool          `mapstructure:"validate-components"`
	ExcludeDeprecated       bool          `mapstructure:"exclude-deprecated"`
	
	// Zarf CLI configuration
	ZarfExtraArgs           string        `mapstructure:"zarf-extra-args"`
	ZarfLintExtraArgs       string        `mapstructure:"zarf-lint-extra-args"`
	ZarfBuildExtraArgs      string        `mapstructure:"zarf-build-extra-args"`
	ZarfDeployExtraArgs     string        `mapstructure:"zarf-deploy-extra-args"`
	
	// Deployment testing configuration
	Upgrade                 bool          `mapstructure:"upgrade"`
	SkipCleanUp             bool          `mapstructure:"skip-clean-up"`
	Namespace               string        `mapstructure:"namespace"`
	DeploymentTimeout       time.Duration `mapstructure:"deployment-timeout"`
	TestTimeout             time.Duration `mapstructure:"test-timeout"`
	KubectlTimeout          time.Duration `mapstructure:"kubectl-timeout"`
	PrintLogs               bool          `mapstructure:"print-logs"`
	
	// Legacy chart-testing compatibility (kept for migration)
	ChartDirs               []string      `mapstructure:"chart-dirs"`
	Charts                  []string      `mapstructure:"charts"`
	ExcludedCharts          []string      `mapstructure:"excluded-charts"`
}

func LoadConfiguration(cfgFile string, cmd *cobra.Command, printConfig bool) (*Configuration, error) {
	v := viper.New()

	// Set Zarf-specific defaults
	v.SetDefault("kubectl-timeout", 30*time.Second)
	v.SetDefault("deployment-timeout", 10*time.Minute)
	v.SetDefault("test-timeout", 5*time.Minute)
	v.SetDefault("print-logs", bool(true))
	v.SetDefault("zarf-dirs", []string{"packages"})
	v.SetDefault("remote", "origin")
	v.SetDefault("target-branch", "main")
	v.SetDefault("since", "HEAD")
	v.SetDefault("check-version-increment", true)
	v.SetDefault("validate-image-pinning", true)
	v.SetDefault("validate-package-schema", true)
	v.SetDefault("validate-components", true)

	cmd.Flags().VisitAll(func(flag *flag.Flag) {
		flagName := flag.Name
		if flagName != "config" && flagName != "help" {
			if err := v.BindPFlag(flagName, flag); err != nil {
				// can't really happen
				panic(fmt.Sprintf("failed binding flag %q: %v\n", flagName, err.Error()))
			}
		}
	})

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.SetEnvPrefix("ZT") // Use ZT prefix for Zarf Testing

	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed loading config file: %w", err)
		}
		if printConfig {
			fmt.Fprintln(os.Stderr, "Using config file:", v.ConfigFileUsed())
		}
	} else {
		// Look for both zt and ct config files for backward compatibility
		configNames := []string{"zt", "zarf-testing", "ct"}
		var configFound bool
		
		for _, configName := range configNames {
			v.SetConfigName(configName)
			
			if cfgDir, ok := os.LookupEnv("ZT_CONFIG_DIR"); ok {
				v.AddConfigPath(cfgDir)
			} else if cfgDir, ok := os.LookupEnv("CT_CONFIG_DIR"); ok {
				// Legacy support
				v.AddConfigPath(cfgDir)
			} else {
				for _, searchLocation := range configSearchLocations {
					v.AddConfigPath(searchLocation)
				}
			}
			
			if err := v.ReadInConfig(); err == nil {
				configFound = true
				if printConfig {
					fmt.Fprintln(os.Stderr, "Using config file:", v.ConfigFileUsed())
				}
				break
			}
		}
		
		if !configFound {
			// No config file found, proceed with defaults
			v.SetConfigName("zt")
		}
	}

	isInstall := strings.Contains(cmd.Use, "install")

	cfg := &Configuration{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed unmarshaling configuration: %w", err)
	}

	// Zarf-specific validation
	if cfg.ProcessAllPackages && len(cfg.Packages) > 0 {
		return nil, errors.New("specifying both, '--all' and '--packages', is not allowed")
	}
	
	// Legacy chart-testing validation for backward compatibility (remove ProcessAllCharts)
	if len(cfg.Charts) > 0 && cfg.ProcessAllPackages {
		return nil, errors.New("specifying both, '--all' and '--charts', is not allowed")
	}

	// Migrate chart-testing config to Zarf config if needed
	if len(cfg.ZarfDirs) == 0 && len(cfg.ChartDirs) > 0 {
		cfg.ZarfDirs = cfg.ChartDirs
	}
	if len(cfg.Packages) == 0 && len(cfg.Charts) > 0 {
		cfg.Packages = cfg.Charts
	}
	if len(cfg.ExcludedPackages) == 0 && len(cfg.ExcludedCharts) > 0 {
		cfg.ExcludedPackages = cfg.ExcludedCharts
	}

	// Disable upgrade (this does some expensive dependency building on previous revisions)
	// when neither "install" nor "lint-and-install" have not been specified.
	cfg.Upgrade = isInstall && cfg.Upgrade
	if (cfg.TargetBranch == "" || cfg.Remote == "") && cfg.Upgrade {
		return nil, errors.New("specifying '--upgrade=true' without '--target-branch' or '--remote', is not allowed")
	}

	// Zarf-specific configuration handling
	if len(cfg.Packages) > 0 || cfg.ProcessAllPackages {
		fmt.Fprintln(os.Stderr, "Version increment checking disabled for specific packages.")
		cfg.CheckVersionIncrement = false
	}
	
	// Legacy support: disable version checking for charts too
	if len(cfg.Charts) > 0 {
		fmt.Fprintln(os.Stderr, "Version increment checking disabled.")
		cfg.CheckVersionIncrement = false
	}

	if printConfig {
		printCfg(cfg)
	}

	return cfg, nil
}

func printCfg(cfg *Configuration) {
	if !cfg.GithubGroups {
		util.PrintDelimiterLineToWriter(os.Stderr, "-")
		fmt.Fprintln(os.Stderr, " Configuration")
		util.PrintDelimiterLineToWriter(os.Stderr, "-")
	} else {
		util.GithubGroupsBegin(os.Stderr, "Configuration")
	}

	e := reflect.ValueOf(cfg).Elem()
	typeOfCfg := e.Type()

	for i := 0; i < e.NumField(); i++ {
		var pattern string
		switch e.Field(i).Kind() {
		case reflect.Bool:
			pattern = "%s: %t\n"
		default:
			pattern = "%s: %s\n"
		}
		fmt.Fprintf(os.Stderr, pattern, typeOfCfg.Field(i).Name, e.Field(i).Interface())
	}

	if !cfg.GithubGroups {
		util.PrintDelimiterLineToWriter(os.Stderr, "-")
	} else {
		util.GithubGroupsEnd(os.Stderr)
	}
}

func findConfigFile(fileName string) (string, error) {
	if dir, ok := os.LookupEnv("CT_CONFIG_DIR"); ok {
		return filepath.Join(dir, fileName), nil
	}

	for _, location := range configSearchLocations {
		filePath := filepath.Join(location, fileName)
		if util.FileExists(filePath) {
			return filePath, nil
		}
	}

	return "", fmt.Errorf("config file not found: %s", fileName)
}
