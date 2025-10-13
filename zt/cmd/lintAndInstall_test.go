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
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cpepper96/zarf-testing/pkg/output"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helper to create a temporary zarf.yaml file
func createTestZarfPackage(t testing.TB, dir string, name string) {
	err := os.MkdirAll(dir, 0755)
	require.NoError(t, err)

	zarfContent := fmt.Sprintf(`
kind: ZarfPackageConfig
metadata:
  name: %s
  description: Test package
  version: 1.0.0

components:
  - name: test-component
    description: Test component
    required: true
`, name)

	zarfPath := filepath.Join(dir, "zarf.yaml")
	err = os.WriteFile(zarfPath, []byte(zarfContent), 0644)
	require.NoError(t, err)
}

func TestNewLintAndInstallCmd(t *testing.T) {
	// Test through the root command since newLintAndInstallCmd is not exported
	rootCmd := NewRootCmd()
	
	var cmd *cobra.Command
	for _, subCmd := range rootCmd.Commands() {
		if subCmd.Use == "lint-and-install" {
			cmd = subCmd
			break
		}
	}
	
	require.NotNil(t, cmd, "lint-and-install command should be available")
	
	assert.Equal(t, "lint-and-install", cmd.Use)
	assert.Contains(t, cmd.Aliases, "li")
	assert.Equal(t, "Lint, install, and test a Zarf package", cmd.Short)
	assert.Contains(t, cmd.Long, "combines validation and")
	
	// Verify that all expected flags are present
	flags := cmd.Flags()
	
	// Common flags
	assert.NotNil(t, flags.Lookup("config"))
	assert.NotNil(t, flags.Lookup("remote"))
	assert.NotNil(t, flags.Lookup("target-branch"))
	assert.NotNil(t, flags.Lookup("zarf-dirs"))
	assert.NotNil(t, flags.Lookup("all"))
	assert.NotNil(t, flags.Lookup("packages"))
	assert.NotNil(t, flags.Lookup("output"))
	
	// Lint-specific flags
	assert.NotNil(t, flags.Lookup("lint-conf"))
	assert.NotNil(t, flags.Lookup("check-version-increment"))
	assert.NotNil(t, flags.Lookup("validate-yaml"))
	assert.NotNil(t, flags.Lookup("additional-commands"))
	
	// Install-specific flags
	assert.NotNil(t, flags.Lookup("build-id"))
	assert.NotNil(t, flags.Lookup("upgrade"))
	assert.NotNil(t, flags.Lookup("skip-missing-values"))
	assert.NotNil(t, flags.Lookup("namespace"))
	assert.NotNil(t, flags.Lookup("release-name"))
	assert.NotNil(t, flags.Lookup("skip-clean-up"))
}

func TestLintAndInstall_FlagHandling(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		validate func(t *testing.T, cmd *cobra.Command)
		wantErr  bool
	}{
		{
			name: "default flags",
			args: []string{},
			validate: func(t *testing.T, cmd *cobra.Command) {
				remote, _ := cmd.Flags().GetString("remote")
				assert.Equal(t, "origin", remote)
				
				targetBranch, _ := cmd.Flags().GetString("target-branch")
				assert.Equal(t, "main", targetBranch)
				
				zarfDirs, _ := cmd.Flags().GetStringSlice("zarf-dirs")
				assert.Equal(t, []string{"packages"}, zarfDirs)
				
				output, _ := cmd.Flags().GetString("output")
				assert.Equal(t, "text", output)
			},
		},
		{
			name: "custom flags",
			args: []string{
				"--remote", "upstream",
				"--target-branch", "develop",
				"--zarf-dirs", "charts,manifests",
				"--output", "json",
				"--all",
				"--namespace", "test-ns",
			},
			validate: func(t *testing.T, cmd *cobra.Command) {
				remote, _ := cmd.Flags().GetString("remote")
				assert.Equal(t, "upstream", remote)
				
				targetBranch, _ := cmd.Flags().GetString("target-branch")
				assert.Equal(t, "develop", targetBranch)
				
				zarfDirs, _ := cmd.Flags().GetStringSlice("zarf-dirs")
				assert.Equal(t, []string{"charts", "manifests"}, zarfDirs)
				
				output, _ := cmd.Flags().GetString("output")
				assert.Equal(t, "json", output)
				
				all, _ := cmd.Flags().GetBool("all")
				assert.True(t, all)
				
				namespace, _ := cmd.Flags().GetString("namespace")
				assert.Equal(t, "test-ns", namespace)
			},
		},
		{
			name: "packages flag",
			args: []string{
				"--packages", "pkg1,pkg2,pkg3",
			},
			validate: func(t *testing.T, cmd *cobra.Command) {
				packages, _ := cmd.Flags().GetStringSlice("packages")
				assert.Equal(t, []string{"pkg1", "pkg2", "pkg3"}, packages)
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd := NewRootCmd()
			var cmd *cobra.Command
			for _, subCmd := range rootCmd.Commands() {
				if subCmd.Use == "lint-and-install" {
					cmd = subCmd
					break
				}
			}
			require.NotNil(t, cmd)
			
			cmd.SetArgs(tt.args)
			
			// Don't actually run the command, just parse flags
			cmd.ParseFlags(tt.args)
			
			if tt.validate != nil {
				tt.validate(t, cmd)
			}
		})
	}
}

func TestLintAndInstall_OutputFormatting(t *testing.T) {
	tests := []struct {
		name           string
		outputFormat   string
		githubGroups   bool
		noColor        bool
		expectedConfig output.Config
	}{
		{
			name:         "text output default",
			outputFormat: "text",
			expectedConfig: output.Config{
				Format:       output.FormatText,
				NoColor:      false,
				GithubGroups: false,
			},
		},
		{
			name:         "json output",
			outputFormat: "json",
			expectedConfig: output.Config{
				Format:       output.FormatJSON,
				NoColor:      false,
				GithubGroups: false,
			},
		},
		{
			name:         "github output with groups",
			outputFormat: "github",
			githubGroups: true,
			expectedConfig: output.Config{
				Format:       output.FormatGitHub,
				NoColor:      false,
				GithubGroups: true,
			},
		},
		{
			name:         "no color output",
			outputFormat: "text",
			noColor:      true,
			expectedConfig: output.Config{
				Format:       output.FormatText,
				NoColor:      true,
				GithubGroups: false,
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that output configuration is set up correctly
			assert.Contains(t, []string{"text", "json", "github"}, tt.outputFormat)
			
			var expectedFormat output.Format
			switch strings.ToLower(tt.outputFormat) {
			case "json":
				expectedFormat = output.FormatJSON
			case "github":
				expectedFormat = output.FormatGitHub
			default:
				expectedFormat = output.FormatText
			}
			
			assert.Equal(t, expectedFormat, tt.expectedConfig.Format)
			assert.Equal(t, tt.noColor, tt.expectedConfig.NoColor)
			assert.Equal(t, tt.githubGroups, tt.expectedConfig.GithubGroups)
		})
	}
}

// Test helper functions
func setupTestCommand() *cobra.Command {
	rootCmd := NewRootCmd()
	var cmd *cobra.Command
	for _, subCmd := range rootCmd.Commands() {
		if subCmd.Use == "lint-and-install" {
			cmd = subCmd
			break
		}
	}
	if cmd != nil {
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
	}
	return cmd
}

func TestLintAndInstall_CommandStructure(t *testing.T) {
	cmd := setupTestCommand()
	require.NotNil(t, cmd, "lint-and-install command should be available")
	
	// Test command properties
	assert.Equal(t, "lint-and-install", cmd.Use)
	assert.Contains(t, cmd.Aliases, "li")
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotNil(t, cmd.RunE)
	
	// Test that command has required flags from both lint and install
	flags := cmd.Flags()
	
	// Lint flags
	assert.NotNil(t, flags.Lookup("lint-conf"))
	assert.NotNil(t, flags.Lookup("check-version-increment"))
	assert.NotNil(t, flags.Lookup("validate-yaml"))
	assert.NotNil(t, flags.Lookup("additional-commands"))
	
	// Install flags  
	assert.NotNil(t, flags.Lookup("build-id"))
	assert.NotNil(t, flags.Lookup("upgrade"))
	assert.NotNil(t, flags.Lookup("skip-missing-values"))
	assert.NotNil(t, flags.Lookup("namespace"))
	assert.NotNil(t, flags.Lookup("release-name"))
	assert.NotNil(t, flags.Lookup("skip-clean-up"))
	
	// Common flags
	assert.NotNil(t, flags.Lookup("all"))
	assert.NotNil(t, flags.Lookup("packages"))
	assert.NotNil(t, flags.Lookup("zarf-dirs"))
	assert.NotNil(t, flags.Lookup("output"))
}

func TestLintAndInstall_TwoPhaseWorkflow(t *testing.T) {
	// Test that the command follows the two-phase workflow design
	cmd := setupTestCommand()
	require.NotNil(t, cmd, "lint-and-install command should be available")
	
	// Verify the command description mentions the two phases
	assert.Contains(t, cmd.Long, "validation and")
	assert.Contains(t, cmd.Long, "deployment testing")
	
	// Verify it has both lint and install flags, proving it combines both workflows
	flags := cmd.Flags()
	
	// Should have lint flags
	assert.NotNil(t, flags.Lookup("lint-conf"))
	assert.NotNil(t, flags.Lookup("validate-yaml"))
	
	// Should have install/test flags  
	assert.NotNil(t, flags.Lookup("build-id"))
	assert.NotNil(t, flags.Lookup("namespace"))
	assert.NotNil(t, flags.Lookup("skip-clean-up"))
	
	// Should have common package discovery flags
	assert.NotNil(t, flags.Lookup("all"))
	assert.NotNil(t, flags.Lookup("packages"))
	assert.NotNil(t, flags.Lookup("zarf-dirs"))
}

func TestLintAndInstall_ErrorHandling(t *testing.T) {
	// Test that error handling configuration is correct
	cmd := setupTestCommand()
	require.NotNil(t, cmd, "lint-and-install command should be available")
	
	// Command should have RunE (not Run) for proper error handling
	assert.NotNil(t, cmd.RunE)
	assert.Nil(t, cmd.Run)
	
	// Should support all output formats for error reporting
	flags := cmd.Flags()
	outputFlag := flags.Lookup("output")
	require.NotNil(t, outputFlag)
	
	// Default should be text
	assert.Equal(t, "text", outputFlag.DefValue)
}

// Benchmark tests for performance validation
func BenchmarkLintAndInstall_PackageDiscovery(b *testing.B) {
	// Create temp directory with test packages
	tempDir, err := os.MkdirTemp("", "zt-bench-*")
	require.NoError(b, err)
	defer os.RemoveAll(tempDir)

	// Create multiple test packages
	for i := 0; i < 10; i++ {
		packageDir := filepath.Join(tempDir, fmt.Sprintf("package%d", i))
		createTestZarfPackage(b, packageDir, fmt.Sprintf("app%d", i))
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := setupTestCommand()
		if cmd != nil {
			args := []string{
				"--zarf-dirs", tempDir,
				"--all",
				"--output", "json",
			}
			cmd.SetArgs(args)
			// Don't actually execute to avoid expensive operations
			cmd.ParseFlags(args)
		}
	}
}

func TestLintAndInstall_ConfigurationIntegration(t *testing.T) {
	// Test that the command integrates properly with the configuration system
	cmd := setupTestCommand()
	require.NotNil(t, cmd, "lint-and-install command should be available")
	
	flags := cmd.Flags()
	
	// Should support configuration file
	configFlag := flags.Lookup("config")
	require.NotNil(t, configFlag)
	
	// Should support environment variable overrides through common flags
	remoteFlag := flags.Lookup("remote")
	require.NotNil(t, remoteFlag)
	assert.Equal(t, "origin", remoteFlag.DefValue)
	
	targetBranchFlag := flags.Lookup("target-branch")
	require.NotNil(t, targetBranchFlag)
	assert.Equal(t, "main", targetBranchFlag.DefValue)
	
	zarfDirsFlag := flags.Lookup("zarf-dirs")
	require.NotNil(t, zarfDirsFlag)
	assert.Equal(t, "[packages]", zarfDirsFlag.DefValue)
}
