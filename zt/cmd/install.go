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

package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/cpepper96/zarf-testing/pkg/config"
	"github.com/cpepper96/zarf-testing/pkg/output"
	"github.com/cpepper96/zarf-testing/pkg/zarf"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
)

func newInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install and test a Zarf package",
		Long: heredoc.Doc(`
			Deploy and test Zarf packages on

			* changed packages (default)
			* specific packages (--packages)
			* all packages (--all)

			in given package directories. This command will deploy packages
			and validate that all components are working correctly.

			Packages will be deployed to test namespaces and validated
			for proper functionality. If no test configuration is present,
			the package is deployed and tested with defaults.`),
		RunE: install,
	}

	flags := cmd.Flags()
	addInstallFlags(flags)
	addCommonLintAndInstallFlags(flags)
	return cmd
}

func addInstallFlags(flags *flag.FlagSet) {
	flags.String("build-id", "", heredoc.Doc(`
		An optional, arbitrary identifier that is added to the name of the namespace a
		chart is installed into. In a CI environment, this could be the build number or
		the ID of a pull request. If not specified, the name of the chart is used`))
	flags.Bool("upgrade", false, heredoc.Doc(`
		Whether to test an in-place upgrade of each chart from its previous revision if the
		current version should not introduce a breaking change according to the SemVer spec`))
	flags.Bool("skip-missing-values", false, heredoc.Doc(`
		When --upgrade has been passed, this flag will skip testing CI values files from the
		previous chart revision if they have been deleted or renamed at the current chart
		revision`))
	flags.String("namespace", "", heredoc.Doc(`
		Namespace to install the release(s) into. If not specified, each release will be
		installed in its own randomly generated namespace`))
	flags.String("release-name", "", heredoc.Doc(`
		Name for the release. If not specified, is set to the chart name and a random 
		identifier.`))
	flags.Bool("skip-clean-up", false, "Skip resources clean-up after testing")
	

}

func install(cmd *cobra.Command, _ []string) error {
	// Setup output formatter
	outputFormat, _ := cmd.Flags().GetString("output")
	noColor, _ := cmd.Flags().GetBool("no-color")
	githubGroups, _ := cmd.Flags().GetBool("github-groups")
	
	var format output.Format
	switch strings.ToLower(outputFormat) {
	case "json":
		format = output.FormatJSON
	case "github":
		format = output.FormatGitHub
	default:
		format = output.FormatText
	}
	
	formatter := output.NewFormatter(&output.Config{
		Format:       format,
		NoColor:      noColor,
		GithubGroups: githubGroups,
		Writer:       os.Stdout,
	})
	
	formatter.Section("Zarf Package Deployment Testing")
	
	// Load configuration
	configuration, err := config.LoadConfiguration("", cmd, false)
	if err != nil {
		formatter.Error("Failed to load configuration: %v", err)
		if format == output.FormatJSON {
			formatter.PrintJSON()
		}
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Determine which packages to test
	var packagesToTest []string
	all, _ := cmd.Flags().GetBool("all")
	packages, _ := cmd.Flags().GetStringSlice("packages")
	
	// Use ZarfDirs with fallback to ChartDirs for backward compatibility
	dirs := configuration.ZarfDirs
	if len(dirs) == 0 {
		dirs = configuration.ChartDirs
	}
	if len(dirs) == 0 {
		dirs = []string{"packages"} // fallback default
	}

	if all {
		formatter.Progress("Finding all packages...")
		allPackages, err := zarf.FindZarfPackages(dirs)
		if err != nil {
			formatter.Error("Failed to find packages: %v", err)
			if format == output.FormatJSON {
				formatter.PrintJSON()
			}
			return fmt.Errorf("failed to find packages: %w", err)
		}
		packagesToTest = allPackages
	} else if len(packages) > 0 {
		formatter.Info("Testing specified packages: %v", packages)
		// Validate that specified packages exist
		for _, pkg := range packages {
			if !zarf.IsZarfPackage(pkg) {
				formatter.Error("Package not found: %s", pkg)
				if format == output.FormatJSON {
					formatter.PrintJSON()
				}
				return fmt.Errorf("package not found: %s", pkg)
			}
		}
		packagesToTest = packages
	} else {
		formatter.Progress("Finding changed packages...")
		changedPackages, err := zarf.FindChangedPackages(configuration.Remote, configuration.TargetBranch, dirs)
		if err != nil {
			formatter.Error("Failed to find changed packages: %v", err)
			if format == output.FormatJSON {
				formatter.PrintJSON()
			}
			return fmt.Errorf("failed to find changed packages: %w", err)
		}
		packagesToTest = changedPackages
	}

	if len(packagesToTest) == 0 {
		formatter.Success("No packages to test")
		if format == output.FormatJSON {
			formatter.PrintJSON()
		}
		return nil
	}

	formatter.Info("Testing %d packages: %v", len(packagesToTest), packagesToTest)

	// Initialize deployer
	deployer, err := zarf.NewDeployer(configuration)
	if err != nil {
		formatter.Error("Failed to initialize deployer: %v", err)
		if format == output.FormatJSON {
			formatter.PrintJSON()
		}
		return fmt.Errorf("failed to initialize deployer: %w", err)
	}

	// Create progress bar for package testing
	progressBar := formatter.NewProgressBar("Testing packages", len(packagesToTest))
	
	// Test each package
	overallSuccess := true
	for i, packagePath := range packagesToTest {
		formatter.Step(i+1, len(packagesToTest), "Testing package: %s", packagePath)
		progressBar.Update(i, fmt.Sprintf("Testing %s", packagePath))
		
		result, err := deployer.TestPackage(packagePath)
		if err != nil {
			formatter.Error("Package %s failed: %v", packagePath, err)
			overallSuccess = false
			continue
		}

		if result.Success {
			formatter.Success("Package %s passed all tests", packagePath)
		} else {
			formatter.Error("Package %s failed validation", packagePath)
			for _, testResult := range result.ComponentTests {
				if !testResult.Success {
					formatter.Warning("  - %s: %s", testResult.ComponentName, testResult.Message)
				}
			}
			overallSuccess = false
		}
	}

	progressBar.Finish("Testing complete")
	formatter.EndSection()
	
	formatter.Section("Results")
	
	if overallSuccess {
		formatter.Success("All packages passed deployment testing")
	} else {
		formatter.Error("Some packages failed deployment testing")
	}
	
	formatter.EndSection()
	
	// Output JSON if requested
	if format == output.FormatJSON {
		if err := formatter.PrintJSON(); err != nil {
			return fmt.Errorf("failed to output JSON: %w", err)
		}
	}
	
	if !overallSuccess {
		os.Exit(1)
	}
	
	return nil
}
