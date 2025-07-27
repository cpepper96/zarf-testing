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
	"os"
	"path/filepath"
	"strings"

	"github.com/cpepper96/zarf-testing/pkg/util"
)

// ZarfPackage represents a Zarf package with its metadata
type ZarfPackage struct {
	Path     string
	Name     string
	Metadata *util.ZarfYaml
}

// FindZarfPackages discovers Zarf packages in the specified directories
func FindZarfPackages(dirs []string) ([]string, error) {
	var packageDirs []string
	
	for _, dir := range dirs {
		packages, err := findPackagesInDirectory(dir)
		if err != nil {
			return nil, fmt.Errorf("failed to find packages in directory %s: %w", dir, err)
		}
		packageDirs = append(packageDirs, packages...)
	}
	
	return packageDirs, nil
}

// findPackagesInDirectory finds all directories containing zarf.yaml files
func findPackagesInDirectory(dir string) ([]string, error) {
	var packages []string
	
	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// Directory doesn't exist, return empty list (not an error)
		return packages, nil
	}
	
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Check if this is a zarf.yaml file
		if info.Name() == "zarf.yaml" && !info.IsDir() {
			packageDir := filepath.Dir(path)
			packages = append(packages, packageDir)
		}
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("error walking directory %s: %w", dir, err)
	}
	
	return packages, nil
}

// IsZarfPackage checks if a directory contains a valid Zarf package
func IsZarfPackage(dir string) bool {
	zarfYamlPath := filepath.Join(dir, "zarf.yaml")
	if _, err := os.Stat(zarfYamlPath); os.IsNotExist(err) {
		return false
	}
	return true
}

// LoadZarfPackage loads a Zarf package from the given directory
func LoadZarfPackage(dir string) (*ZarfPackage, error) {
	if !IsZarfPackage(dir) {
		return nil, fmt.Errorf("directory %s does not contain a zarf.yaml file", dir)
	}
	
	zarfYamlPath := filepath.Join(dir, "zarf.yaml")
	metadata, err := util.ReadZarfYaml(zarfYamlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read zarf.yaml from %s: %w", zarfYamlPath, err)
	}
	
	// Extract package name from metadata or use directory name as fallback
	packageName := metadata.Metadata.Name
	if packageName == "" {
		packageName = filepath.Base(dir)
	}
	
	return &ZarfPackage{
		Path:     dir,
		Name:     packageName,
		Metadata: metadata,
	}, nil
}

// FilterExcludedPackages removes excluded packages from the list
func FilterExcludedPackages(packages []string, excluded []string) []string {
	if len(excluded) == 0 {
		return packages
	}
	
	var filtered []string
	excludeSet := make(map[string]bool)
	
	// Create set of excluded package names/patterns
	for _, exc := range excluded {
		excludeSet[exc] = true
	}
	
	for _, pkg := range packages {
		packageName := filepath.Base(pkg)
		if !excludeSet[packageName] && !excludeSet[pkg] {
			filtered = append(filtered, pkg)
		}
	}
	
	return filtered
}

// ValidatePackages validates that all package directories contain valid Zarf packages
func ValidatePackages(packageDirs []string) error {
	var errors []string
	
	for _, dir := range packageDirs {
		if !IsZarfPackage(dir) {
			errors = append(errors, fmt.Sprintf("directory %s does not contain a valid Zarf package", dir))
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("package validation failed:\n- %s", strings.Join(errors, "\n- "))
	}
	
	return nil
}
