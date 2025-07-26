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

package zarf

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cpepper96/zarf-testing/pkg/exec"
	"github.com/cpepper96/zarf-testing/pkg/tool"
)

// FindChangedPackages identifies Zarf packages that have been changed between Git references
func FindChangedPackages(remote, targetBranch string, dirs []string) ([]string, error) {
	executor := exec.NewProcessExecutor(false) // debug = false
	git := tool.NewGit(executor)
	
	// Get list of changed files using merge base
	mergeBase, err := git.MergeBase(fmt.Sprintf("%s/%s", remote, targetBranch), "HEAD")
	if err != nil {
		return nil, fmt.Errorf("failed to get merge base: %w", err)
	}
	
	changedFiles, err := git.ListChangedFilesInDirs(mergeBase, dirs...)
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}
	
	// Find packages containing changed files
	changedPackages := make(map[string]bool)
	
	for _, file := range changedFiles {
		packageDir, err := findPackageContainingFile(file, dirs)
		if err != nil {
			continue // Skip files that aren't in Zarf packages
		}
		if packageDir != "" {
			changedPackages[packageDir] = true
		}
	}
	
	// Convert map to slice
	var result []string
	for pkg := range changedPackages {
		result = append(result, pkg)
	}
	
	return result, nil
}

// findPackageContainingFile finds the Zarf package directory that contains the given file
func findPackageContainingFile(file string, dirs []string) (string, error) {
	// Walk up the directory tree to find a zarf.yaml file
	currentDir := filepath.Dir(file)
	
	for currentDir != "." && currentDir != "/" {
		// Check if this directory contains a zarf.yaml
		if IsZarfPackage(currentDir) {
			// Verify this package is in one of the configured directories
			for _, dir := range dirs {
				if strings.HasPrefix(currentDir, dir) {
					return currentDir, nil
				}
			}
		}
		currentDir = filepath.Dir(currentDir)
	}
	
	return "", fmt.Errorf("file %s is not in a Zarf package", file)
}

// GetChangedFilesMatchingPattern gets changed files that match a specific pattern (e.g., "zarf.yaml")
func GetChangedFilesMatchingPattern(remote, targetBranch, pattern string) ([]string, error) {
	executor := exec.NewProcessExecutor(false) // debug = false
	git := tool.NewGit(executor)
	
	// Get merge base and then changed files
	mergeBase, err := git.MergeBase(fmt.Sprintf("%s/%s", remote, targetBranch), "HEAD")
	if err != nil {
		return nil, fmt.Errorf("failed to get merge base: %w", err)
	}
	
	allChangedFiles, err := git.ListChangedFilesInDirs(mergeBase, ".")
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}
	
	var matchingFiles []string
	for _, file := range allChangedFiles {
		if filepath.Base(file) == pattern {
			matchingFiles = append(matchingFiles, file)
		}
	}
	
	return matchingFiles, nil
}
