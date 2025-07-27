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

	"github.com/MakeNowJust/heredoc"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	cfgFile string
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "zt",
		Short: "The Zarf package testing tool",
		Long: heredoc.Doc(`
			Lint and test

			* changed packages
			* specific packages
			* all packages

			in given package directories.`),
		SilenceUsage: true,
	}

	cmd.AddCommand(newLintCmd())
	cmd.AddCommand(newInstallCmd())
	cmd.AddCommand(newLintAndInstallCmd())
	cmd.AddCommand(newListChangedCmd())
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newGenerateDocsCmd())

	cmd.DisableAutoGenTag = true

	return cmd
}

// Execute runs the application
func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func addCommonFlags(flags *pflag.FlagSet) {
	flags.StringVar(&cfgFile, "config", "", "Config file")
	flags.String("remote", "origin", "The name of the Git remote used to identify changed charts")
	flags.String("target-branch", "main", "The name of the target branch used to identify changed packages")
	flags.String("since", "HEAD", "The Git reference used to identify changed packages")
	flags.StringSlice("zarf-dirs", []string{"packages"}, heredoc.Doc(`
		Directories containing Zarf packages. May be specified multiple times
		or separate values with commas`))
	flags.StringSlice("excluded-packages", []string{}, heredoc.Doc(`
		Packages that should be skipped. May be specified multiple times
		or separate values with commas`))
	flags.Bool("print-config", false, "Prints the configuration to stderr")
	flags.Bool("exclude-deprecated", false, "Skip packages that are marked as deprecated")
	flags.Bool("github-groups", false, heredoc.Doc(`
		Change the delimiters for github to create collapsible groups
		for command output`))
	
	// Output formatting flags
	flags.String("output", "text", "Output format: text, json, github")
	flags.Bool("no-color", false, "Disable colored output")
}

func addCommonLintAndInstallFlags(flags *pflag.FlagSet) {
	addCommonFlags(flags)
	flags.Bool("all", false, heredoc.Doc(`
		Process all packages except those explicitly excluded.
		Disables changed package detection and version increment checking`))
	flags.StringSlice("packages", []string{}, heredoc.Doc(`
		Specific packages to test. Disables changed package detection and
		version increment checking. May be specified multiple times
		or separate values with commas`))


	flags.Bool("debug", false, "Print CLI calls of external tools to stdout")
}
