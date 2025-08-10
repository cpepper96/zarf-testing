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
)

func newLintAndInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "lint-and-install",
		Aliases: []string{"li"},
		Short:   "Lint, install, and test a Zarf package",
		Long: heredoc.Doc(`
			Lint, deploy and test Zarf packages on

			* changed packages (default)
			* specific packages (--packages)
			* all packages (--all)

			in given package directories. This command combines validation and
			deployment testing into a single workflow.

			Packages are first validated for proper structure and configuration,
			then deployed to test namespaces and validated for functionality.
			If validation fails, deployment is skipped.`),
		RunE: lintAndInstall,
	}

	flags := cmd.Flags()
	addLintFlags(flags)
	addInstallFlags(flags)
	addCommonLintAndInstallFlags(flags)
	return cmd
}

func lintAndInstall(cmd *cobra.Command, _ []string) error {
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
	
	formatter.Section("Zarf Package Lint and Install Testing")
	
	// Load configuration
	configuration, err := config.LoadConfiguration("", cmd, false)
	if err != nil {
		formatter.Error("Failed to load configuration: %v", err)
		if format == output.FormatJSON {
			formatter.PrintJSON()
		}
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Determine which packages to process
	var packageDirs []string
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

	if len(packages) > 0 {
		// Specific packages specified
		packageDirs = packages
		formatter.Info("Processing specified packages: %v", packages)
	} else if all {
		// Process all packages
		formatter.Progress("Finding all packages...")
		packageDirs, err = zarf.FindZarfPackages(dirs)
		if err != nil {
			formatter.Error("Failed to find packages: %v", err)
			if format == output.FormatJSON {
				formatter.PrintJSON()
			}
			return fmt.Errorf("failed to find packages: %w", err)
		}
		formatter.Info("Processing all packages in directories: %v", dirs)
	} else {
		// Default: process changed packages
		formatter.Progress("Finding changed packages...")
		packageDirs, err = zarf.FindChangedPackages(configuration.Remote, configuration.TargetBranch, dirs)
		if err != nil {
			formatter.Error("Failed to find changed packages: %v", err)
			if format == output.FormatJSON {
				formatter.PrintJSON()
			}
			return fmt.Errorf("failed to find changed packages: %w", err)
		}
		
		if len(packageDirs) == 0 {
			formatter.Success("No changed packages found")
			if format == output.FormatJSON {
				formatter.PrintJSON()
			}
			return nil
		}
		formatter.Info("Processing changed packages: %v", packageDirs)
	}

	if len(packageDirs) == 0 {
		formatter.Success("No packages to process")
		if format == output.FormatJSON {
			formatter.PrintJSON()
		}
		return nil
	}

	formatter.EndSection()

	// Phase 1: Validation
	formatter.Section("Phase 1: Package Validation")
	formatter.Info("Validating %d packages", len(packageDirs))

	// Create validator
	validator := zarf.NewPackageValidator()
	
	// Validate packages
	results, err := validator.ValidatePackages(packageDirs)
	if err != nil {
		formatter.Error("Failed to validate packages: %v", err)
		if format == output.FormatJSON {
			formatter.PrintJSON()
		}
		return fmt.Errorf("failed to validate packages: %w", err)
	}
	
	// Print validation results
	zarf.PrintValidationResults(results)
	
	// Check if there were any validation errors
	if zarf.HasValidationErrors(results) {
		formatter.Error("Package validation failed - skipping deployment phase")
		formatter.EndSection()
		if format == output.FormatJSON {
			formatter.PrintJSON()
		}
		return fmt.Errorf("package validation failed")
	}
	
	formatter.Success("All packages passed validation")
	formatter.EndSection()

	// Phase 2: Deployment Testing
	formatter.Section("Phase 2: Package Deployment Testing")
	formatter.Info("Testing deployment of %d packages", len(packageDirs))

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
	progressBar := formatter.NewProgressBar("Testing packages", len(packageDirs))
	
	// Test each package
	overallSuccess := true
	for i, packagePath := range packageDirs {
		formatter.Step(i+1, len(packageDirs), "Testing package: %s", packagePath)
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
	
	// Final Results
	formatter.Section("Final Results")
	
	if overallSuccess {
		formatter.Success("All packages passed lint and install testing")
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
