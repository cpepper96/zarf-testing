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
)

func newListChangedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list-changed",
		Aliases: []string{"ls-changed", "lsc"},
		Short:   "List changed packages",
		Long: heredoc.Doc(`
			"List changed Zarf packages based on configured package directories,
			"remote, and target branch`),
		RunE: listChanged,
	}

	flags := cmd.Flags()
	addCommonFlags(flags)
	return cmd
}

func listChanged(cmd *cobra.Command, _ []string) error {
	// Get basic flags for package discovery
	remote, err := cmd.Flags().GetString("remote")
	if err != nil {
		return err
	}
	
	targetBranch, err := cmd.Flags().GetString("target-branch")
	if err != nil {
		return err
	}
	
	zarfDirs, err := cmd.Flags().GetStringSlice("zarf-dirs")
	if err != nil {
		return err
	}
	
	// Find changed packages
	changedPackages, err := zarf.FindChangedPackages(remote, targetBranch, zarfDirs)
	if err != nil {
		return fmt.Errorf("failed to find changed packages: %w", err)
	}
	
	// Output each changed package directory
	for _, pkg := range changedPackages {
		fmt.Println(pkg)
	}
	
	return nil
}
