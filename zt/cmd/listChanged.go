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
	fmt.Println("Listing changed Zarf packages - NOT IMPLEMENTED YET")
	fmt.Println("This command will detect changed Zarf packages via Git")
	
	// TODO: Implement actual changed package detection
	// 1. Load configuration
	// 2. Use Git to find changed files  
	// 3. Identify packages that contain changed zarf.yaml files
	// 4. Output list of changed package directories
	
	return fmt.Errorf("list-changed command not yet implemented - coming in Task 2.1 (Package Discovery)")
}
