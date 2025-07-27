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

	"github.com/MakeNowJust/heredoc"
	"github.com/cpepper96/zarf-testing/pkg/zarf"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
)

func newLintCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lint",
		Short: "Lint and validate a Zarf package",
		Long: heredoc.Doc(`
			Run Zarf package validation, version checking, YAML schema validation
			on 'zarf.yaml', YAML linting on package configuration files,
			and component validation on

			* changed packages (default)
			* specific packages (--packages)
			* all packages (--all)

			in given package directories.

			Packages may have multiple custom configuration files in the package
			directory. The package is linted and validated according to Zarf
			package specifications and component requirements.`),
		RunE: lint,
	}

	flags := cmd.Flags()
	addLintFlags(flags)
	addCommonLintAndInstallFlags(flags)
	return cmd
}

func addLintFlags(flags *flag.FlagSet) {
	flags.String("lint-conf", "", heredoc.Doc(`
		The config file for YAML linting. If not specified, 'lintconf.yaml'
		is searched in the current directory, '$HOME/.zt', and '/etc/zt', in
		that order`))
	flags.Bool("check-version-increment", true, "Activates a check for package version increments")
	flags.Bool("validate-yaml", true, "Enable linting of 'zarf.yaml' and configuration files")
	flags.StringSlice("additional-commands", []string{}, heredoc.Doc(`
		Additional commands to run per package (default: [])
		Commands will be executed in the same order as provided in the list and will
		be rendered with go template before being executed.
		Example: "zarf package inspect {{ .Path }}"`))
}

func lint(cmd *cobra.Command, _ []string) error {
	fmt.Println("Linting Zarf packages...")
	
	// Get flags for package discovery
	zarfDirs, err := cmd.Flags().GetStringSlice("zarf-dirs")
	if err != nil {
		return err
	}
	
	all, err := cmd.Flags().GetBool("all")
	if err != nil {
		return err
	}
	
	packages, err := cmd.Flags().GetStringSlice("packages")
	if err != nil {
		return err
	}
	
	var packageDirs []string
	
	// Determine which packages to lint
	if len(packages) > 0 {
		// Specific packages specified
		packageDirs = packages
		fmt.Printf("Linting specified packages: %v\n", packages)
	} else if all {
		// Lint all packages
		packageDirs, err = zarf.FindZarfPackages(zarfDirs)
		if err != nil {
			return fmt.Errorf("failed to find packages: %w", err)
		}
		fmt.Printf("Linting all packages in directories: %v\n", zarfDirs)
	} else {
		// Default: lint changed packages
		remote, err := cmd.Flags().GetString("remote")
		if err != nil {
			return err
		}
		targetBranch, err := cmd.Flags().GetString("target-branch")
		if err != nil {
			return err
		}
		
		packageDirs, err = zarf.FindChangedPackages(remote, targetBranch, zarfDirs)
		if err != nil {
			return fmt.Errorf("failed to find changed packages: %w", err)
		}
		
		if len(packageDirs) == 0 {
			fmt.Println("No changed packages found")
			return nil
		}
		fmt.Printf("Linting changed packages: %v\n", packageDirs)
	}
	
	// Create validator
	validator := zarf.NewPackageValidator()
	
	// Validate packages
	results, err := validator.ValidatePackages(packageDirs)
	if err != nil {
		return fmt.Errorf("failed to validate packages: %w", err)
	}
	
	// Print results
	zarf.PrintValidationResults(results)
	
	// Check if there were any errors
	if zarf.HasValidationErrors(results) {
		return fmt.Errorf("package validation failed")
	}
	
	fmt.Println("\nAll packages linted successfully")
	return nil
}
