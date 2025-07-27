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

	"github.com/cpepper96/zarf-testing/pkg/exec"
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
			result.Warnings = append(result.Warnings, fmt.Sprintf("Zarf CLI validation failed, falling back to basic validation: %v", err))
			v.UseSDK = false // Disable SDK for future calls in this session
		} else {
			// Add indicator that we used Zarf CLI validation
			sdkResult.Warnings = append(sdkResult.Warnings, "Validated using Zarf CLI")
			return sdkResult, nil
		}
	}
	
	// Fallback to basic validation
	return v.validateBasic(packagePath)
}

// validateWithSDK attempts to validate using the Zarf CLI wrapper
func (v *PackageValidator) validateWithSDK(packagePath string) (*ValidationResult, error) {
	result := &ValidationResult{
		PackagePath: packagePath,
		Valid:       true,
		Errors:      []string{},
		Warnings:    []string{},
	}
	
	// Try to run zarf dev lint using CLI wrapper
	executor := exec.NewProcessExecutor(false) // debug = false
	
	// Check if zarf CLI is available
	_, err := executor.RunProcessAndCaptureOutput("zarf", "version")
	if err != nil {
		return nil, fmt.Errorf("zarf CLI not found - please install Zarf CLI for full validation: %w", err)
	}
	
	// Run zarf dev lint on the package - we need to capture output even on error
	cmd, err := executor.CreateProcess("zarf", "dev", "lint")
	if err != nil {
		return nil, fmt.Errorf("failed to create zarf process: %w", err)
	}
	
	cmd.Dir = packagePath
	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))
	
	if err != nil {
		// zarf dev lint failed - parse the output for errors
		result.Valid = false
		
		// Parse output for more specific errors
		if outputStr != "" {
			lines := strings.Split(outputStr, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" && !strings.Contains(line, "Using build directory") {
					// Parse Zarf log format (timestamp LEVEL message)
					if strings.Contains(line, " ERR ") {
						// Extract message after "ERR "
						parts := strings.SplitN(line, " ERR ", 2)
						if len(parts) == 2 {
							result.Errors = append(result.Errors, parts[1])
						} else {
							result.Errors = append(result.Errors, line)
						}
					} else if strings.Contains(line, " WRN ") {
						// Extract message after "WRN "
						parts := strings.SplitN(line, " WRN ", 2)
						if len(parts) == 2 {
							result.Warnings = append(result.Warnings, parts[1])
						} else {
							result.Warnings = append(result.Warnings, line)
						}
					} else if strings.Contains(line, "ERROR") || strings.Contains(line, "error") || 
					         strings.Contains(line, "FAIL") || strings.Contains(line, "fail") {
						result.Errors = append(result.Errors, line)
					}
				}
			}
		}
	} else {
		// zarf dev lint succeeded
		result.Valid = true
		
		// Parse output for warnings even on success
		if outputStr != "" {
			lines := strings.Split(outputStr, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" && strings.Contains(line, " WRN ") {
					// Extract message after "WRN "
					parts := strings.SplitN(line, " WRN ", 2)
					if len(parts) == 2 {
						result.Warnings = append(result.Warnings, parts[1])
					} else {
						result.Warnings = append(result.Warnings, line)
					}
				}
			}
		}
	}
	
	// Additional zarf-testing specific validations (beyond what zarf dev lint does)
	versionErr := v.validateVersionIncrement(packagePath, result)
	if versionErr != nil {
		return nil, fmt.Errorf("version increment validation failed: %w", versionErr)
	}
	
	// Add image pinning validation
	imagePinErr := v.validateImagePinning(packagePath, result)
	if imagePinErr != nil {
		return nil, fmt.Errorf("image pinning validation failed: %w", imagePinErr)
	}
	
	return result, nil
}

// validateVersionIncrement checks if package version was incremented when components changed
func (v *PackageValidator) validateVersionIncrement(packagePath string, result *ValidationResult) error {
	// This is the key validation that zarf dev lint doesn't do
	// We need to compare with the previous version from Git
	
	executor := exec.NewProcessExecutor(false)
	
	// Get the current zarf.yaml
	currentZarfPath := filepath.Join(packagePath, "zarf.yaml")
	currentContent, err := util.ReadZarfYaml(currentZarfPath)
	if err != nil {
		return fmt.Errorf("failed to read current zarf.yaml: %w", err)
	}
	
	// Get the previous version from Git (HEAD~1 or target branch)
	previousContent, err := executor.RunProcessAndCaptureOutput("git", "show", "HEAD~1:"+filepath.Join(packagePath, "zarf.yaml"))
	if err != nil {
		// If we can't get previous version, skip this validation
		result.Warnings = append(result.Warnings, "Could not retrieve previous package version for comparison")
		return nil
	}
	
	previousZarf, err := util.UnmarshalZarfYaml([]byte(previousContent))
	if err != nil {
		// If we can't parse previous version, skip this validation  
		result.Warnings = append(result.Warnings, "Could not parse previous package version for comparison")
		return nil
	}
	
	// Compare versions
	if currentContent.Metadata.Version == previousZarf.Metadata.Version {
		// Versions are the same - check if package content changed
		currentYamlStr, _ := executor.RunProcessAndCaptureOutput("cat", currentZarfPath)
		if currentYamlStr != previousContent {
			result.Errors = append(result.Errors, 
				fmt.Sprintf("Package content changed but version not incremented (still %s)", 
				currentContent.Metadata.Version))
			result.Valid = false
		}
	}
	
	return nil
}

// validateImagePinning checks if images are pinned with digests (similar to Zarf's warnings)
func (v *PackageValidator) validateImagePinning(packagePath string, result *ValidationResult) error {
	// Read the zarf.yaml to check for image references
	zarfYamlPath := filepath.Join(packagePath, "zarf.yaml")
	content, err := os.ReadFile(zarfYamlPath)
	if err != nil {
		return fmt.Errorf("failed to read zarf.yaml for image pinning validation: %w", err)
	}
	
	contentStr := string(content)
	
	// Look for image references in the YAML content
	// This is a simplified check - in production you'd parse the YAML structure
	lines := strings.Split(contentStr, "\n")
	inImagesSection := false
	
	for _, line := range lines {
		originalLine := line
		line = strings.TrimSpace(line)
		
		// Track if we're in an images section
		if strings.Contains(line, "images:") {
			inImagesSection = true
			continue
		}
		
		// Reset if we hit a new section at the same or higher level
		if !strings.HasPrefix(originalLine, " ") && !strings.HasPrefix(originalLine, "\t") && strings.Contains(line, ":") {
			inImagesSection = false
		}
		
		// Check for image references in images section
		if inImagesSection && strings.HasPrefix(line, "- ") {
			imageName := strings.TrimPrefix(line, "- ")
			imageName = strings.TrimSpace(imageName)
			imageName = strings.Trim(imageName, "\"'")
			
			// Check if image is pinned with digest
			if strings.Contains(imageName, ":") && !strings.Contains(imageName, "@sha256:") {
				// Skip if it's a variable reference
				if !strings.HasPrefix(imageName, "{{") && !strings.HasPrefix(imageName, "${") {
					result.Warnings = append(result.Warnings, 
						fmt.Sprintf("Image not pinned with digest - %s", imageName))
				}
			}
		}
		
		// Also check for direct image: references
		if strings.Contains(line, "image:") {
			imagePart := strings.Split(line, "image:")[1]
			imagePart = strings.TrimSpace(imagePart)
			imagePart = strings.Trim(imagePart, "\"'")
			
			// Check if image is pinned with digest
			if strings.Contains(imagePart, ":") && !strings.Contains(imagePart, "@sha256:") {
				// Skip if it's a variable reference
				if !strings.HasPrefix(imagePart, "{{") && !strings.HasPrefix(imagePart, "${") {
					result.Warnings = append(result.Warnings, 
						fmt.Sprintf("Image not pinned with digest - %s", imagePart))
				}
			}
		}
	}
	
	return nil
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
