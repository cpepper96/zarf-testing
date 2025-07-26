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

	"github.com/cpepper96/zarf-testing/pkg/util"
)

// ValidationResult represents the result of Zarf package validation
type ValidationResult struct {
	PackagePath string
	Valid       bool
	Errors      []string
	Warnings    []string
}

// PackageValidator handles Zarf package validation
type PackageValidator struct {
	UseSDK bool // Whether to use Zarf SDK or fallback to basic validation
}

// NewPackageValidator creates a new package validator
func NewPackageValidator() *PackageValidator {
	return &PackageValidator{
		UseSDK: true, // Try SDK first, fallback if it fails
	}
}

// ValidatePackage validates a Zarf package at the given path
func (v *PackageValidator) ValidatePackage(packagePath string) (*ValidationResult, error) {
	result := &ValidationResult{
		PackagePath: packagePath,
		Valid:       false,
		Errors:      []string{},
		Warnings:    []string{},
	}
	
	// First check if this is actually a Zarf package
	if !IsZarfPackage(packagePath) {
		result.Errors = append(result.Errors, "Directory does not contain a zarf.yaml file")
		return result, nil
	}
	
	// Try SDK validation first
	if v.UseSDK {
		sdkResult, err := v.validateWithSDK(packagePath)
		if err != nil {
			// SDK failed, log warning and fall back to basic validation
			result.Warnings = append(result.Warnings, fmt.Sprintf("Zarf SDK validation failed, falling back to basic validation: %v", err))
			v.UseSDK = false // Disable SDK for future calls in this session
		} else {
			return sdkResult, nil
		}
	}
	
	// Fallback to basic validation
	return v.validateBasic(packagePath)
}

// validateWithSDK attempts to validate using the Zarf SDK
func (v *PackageValidator) validateWithSDK(packagePath string) (*ValidationResult, error) {
	// TODO: This is where we would implement actual Zarf SDK integration
	// For now, return an error to trigger fallback
	return nil, fmt.Errorf("Zarf SDK integration not yet implemented - coming in next iteration")
}

// validateBasic performs basic validation without the Zarf SDK
func (v *PackageValidator) validateBasic(packagePath string) (*ValidationResult, error) {
	result := &ValidationResult{
		PackagePath: packagePath,
		Valid:       true,
		Errors:      []string{},
		Warnings:    []string{},
	}
	
	// Load and parse the zarf.yaml file
	zarfYamlPath := filepath.Join(packagePath, "zarf.yaml")
	zarfYaml, err := util.ReadZarfYaml(zarfYamlPath)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to parse zarf.yaml: %v", err))
		return result, nil
	}
	
	// Basic validation checks
	if zarfYaml.Kind == "" {
		result.Errors = append(result.Errors, "Missing 'kind' field in zarf.yaml")
		result.Valid = false
	} else if zarfYaml.Kind != "ZarfPackageConfig" {
		result.Errors = append(result.Errors, fmt.Sprintf("Invalid kind '%s', expected 'ZarfPackageConfig'", zarfYaml.Kind))
		result.Valid = false
	}
	
	if zarfYaml.Metadata.Name == "" {
		result.Errors = append(result.Errors, "Missing package name in metadata")
		result.Valid = false
	}
	
	if zarfYaml.Metadata.Version == "" {
		result.Warnings = append(result.Warnings, "No version specified in metadata")
	}
	
	if zarfYaml.Metadata.Description == "" {
		result.Warnings = append(result.Warnings, "No description provided in metadata")
	}
	
	// Check for common naming conventions
	if zarfYaml.Metadata.Name != "" {
		if len(zarfYaml.Metadata.Name) > 63 {
			result.Errors = append(result.Errors, "Package name must be 63 characters or less")
			result.Valid = false
		}
	}
	
	return result, nil
}

// ValidatePackages validates multiple packages and returns results
func (v *PackageValidator) ValidatePackages(packagePaths []string) ([]*ValidationResult, error) {
	var results []*ValidationResult
	
	for _, path := range packagePaths {
		result, err := v.ValidatePackage(path)
		if err != nil {
			return nil, fmt.Errorf("failed to validate package %s: %w", path, err)
		}
		results = append(results, result)
	}
	
	return results, nil
}

// PrintValidationResults prints validation results in a user-friendly format
func PrintValidationResults(results []*ValidationResult) {
	for _, result := range results {
		fmt.Printf("\n==> Linting %s\n", result.PackagePath)
		
		if len(result.Errors) > 0 {
			fmt.Println("[ERROR] Validation failed:")
			for _, err := range result.Errors {
				fmt.Printf("  - %s\n", err)
			}
		}
		
		if len(result.Warnings) > 0 {
			fmt.Println("[WARNING] Issues found:")
			for _, warning := range result.Warnings {
				fmt.Printf("  - %s\n", warning)
			}
		}
		
		if result.Valid && len(result.Warnings) == 0 {
			fmt.Println("[INFO] Package validation successful")
		} else if result.Valid {
			fmt.Println("[INFO] Package validation successful (with warnings)")
		} else {
			fmt.Println("[ERROR] Package validation failed")
		}
	}
}

// HasValidationErrors checks if any of the results have errors
func HasValidationErrors(results []*ValidationResult) bool {
	for _, result := range results {
		if !result.Valid {
			return true
		}
	}
	return false
}
