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
		is searched in the current directory, '$HOME/.ct', and '/etc/ct', in
		that order`))
	flags.String("chart-yaml-schema", "", heredoc.Doc(`
		The schema for chart.yml validation. If not specified, 'chart_schema.yaml'
		is searched in the current directory, '$HOME/.ct', and '/etc/ct', in
		that order.`))
	flags.Bool("validate-maintainers", true, heredoc.Doc(`
		Enable validation of maintainer account names in chart.yml.
		Works for GitHub, GitLab, and Bitbucket`))
	flags.Bool("check-version-increment", true, "Activates a check for chart version increments")
	flags.Bool("validate-chart-schema", true, heredoc.Doc(`
		Enable schema validation of 'Chart.yaml' using Yamale`))
	flags.Bool("validate-yaml", true, heredoc.Doc(`
		Enable linting of 'Chart.yaml' and values files`))
	flags.Bool("skip-helm-dependencies", false, heredoc.Doc(`
		Skip running 'helm dependency build' before linting`))
	flags.StringSlice("additional-commands", []string{}, heredoc.Doc(`
		Additional commands to run per chart (default: [])
		Commands will be executed in the same order as provided in the list and will
		be rendered with go template before being executed.
		Example: "helm unittest --helm3 -f tests/*.yaml {{ .Path }}"`))
}

func lint(cmd *cobra.Command, _ []string) error {
	fmt.Println("Zarf package linting - NOT IMPLEMENTED YET")
	fmt.Println("This command will validate Zarf packages using the Zarf SDK")
	
	// TODO: Implement actual Zarf package linting
	// 1. Load configuration 
	// 2. Discover Zarf packages
	// 3. Validate each package using Zarf SDK
	// 4. Report results
	
	return fmt.Errorf("lint command not yet implemented - coming in Task 2.2 (Zarf SDK Integration)")
}
