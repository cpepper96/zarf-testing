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
	
	// Advanced component validation rules
	componentErr := v.validateComponents(packagePath, result)
	if componentErr != nil {
		return nil, fmt.Errorf("component validation failed: %w", componentErr)
	}
	
	// Validate component dependencies
	depsErr := v.validateComponentDependencies(packagePath, result)
	if depsErr != nil {
		return nil, fmt.Errorf("component dependency validation failed: %w", depsErr)
	}
	
	// Validate security best practices
	securityErr := v.validateSecurityBestPractices(packagePath, result)
	if securityErr != nil {
		return nil, fmt.Errorf("security validation failed: %w", securityErr)
	}
	
	// Validate resource constraints and sizing
	resourceErr := v.validateResourceConstraints(packagePath, result)
	if resourceErr != nil {
		return nil, fmt.Errorf("resource validation failed: %w", resourceErr)
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

// validateComponents performs advanced component validation
func (v *PackageValidator) validateComponents(packagePath string, result *ValidationResult) error {
	zarfYaml, err := util.ReadZarfYaml(filepath.Join(packagePath, "zarf.yaml"))
	if err != nil {
		return fmt.Errorf("failed to read zarf.yaml for component validation: %w", err)
	}
	
	if len(zarfYaml.Components) == 0 {
		result.Warnings = append(result.Warnings, "Package has no components defined")
		return nil
	}
	
	// Check for component naming conventions
	componentNames := make(map[string]bool)
	for _, component := range zarfYaml.Components {
		// Check for duplicate component names
		if componentNames[component.Name] {
			result.Errors = append(result.Errors, fmt.Sprintf("Duplicate component name: %s", component.Name))
		}
		componentNames[component.Name] = true
		
		// Check component naming conventions
		if !isValidComponentName(component.Name) {
			result.Warnings = append(result.Warnings, 
				fmt.Sprintf("Component name '%s' doesn't follow naming conventions (lowercase, hyphens, no spaces)", component.Name))
		}
		
		// Check for required components without default
		if component.Required && component.Default {
			result.Warnings = append(result.Warnings, 
				fmt.Sprintf("Component '%s' is both required and default (redundant)", component.Name))
		}
		
		// Check for empty components
		if len(component.Files) == 0 && len(component.Charts) == 0 && 
		   len(component.Manifests) == 0 && len(component.Images) == 0 && 
		   len(component.Repos) == 0 && len(component.DataInjections) == 0 {
			result.Warnings = append(result.Warnings, 
				fmt.Sprintf("Component '%s' appears to be empty (no files, charts, manifests, images, etc.)", component.Name))
		}
	}
	
	return nil
}

// validateComponentDependencies checks component dependency relationships
func (v *PackageValidator) validateComponentDependencies(packagePath string, result *ValidationResult) error {
	zarfYaml, err := util.ReadZarfYaml(filepath.Join(packagePath, "zarf.yaml"))
	if err != nil {
		return fmt.Errorf("failed to read zarf.yaml for dependency validation: %w", err)
	}
	
	if len(zarfYaml.Components) == 0 {
		return nil
	}
	
	// Build component map for lookup
	componentMap := make(map[string]*util.ZarfComponent)
	for i := range zarfYaml.Components {
		componentMap[zarfYaml.Components[i].Name] = &zarfYaml.Components[i]
	}
	
	// Validate dependencies
	for _, component := range zarfYaml.Components {
		for _, dep := range component.DepsWith {
			// Check if dependency exists
			if _, exists := componentMap[dep]; !exists {
				result.Errors = append(result.Errors, 
					fmt.Sprintf("Component '%s' depends on non-existent component '%s'", component.Name, dep))
			}
			
			// Check for circular dependencies
			if hasDependencyCycle(component.Name, dep, componentMap, make(map[string]bool)) {
				result.Errors = append(result.Errors, 
					fmt.Sprintf("Circular dependency detected between '%s' and '%s'", component.Name, dep))
			}
		}
		
		// Check for self-dependencies
		for _, dep := range component.DepsWith {
			if dep == component.Name {
				result.Errors = append(result.Errors, 
					fmt.Sprintf("Component '%s' cannot depend on itself", component.Name))
			}
		}
	}
	
	return nil
}

// validateSecurityBestPractices checks for security best practices
func (v *PackageValidator) validateSecurityBestPractices(packagePath string, result *ValidationResult) error {
	zarfYaml, err := util.ReadZarfYaml(filepath.Join(packagePath, "zarf.yaml"))
	if err != nil {
		return fmt.Errorf("failed to read zarf.yaml for security validation: %w", err)
	}
	
	for _, component := range zarfYaml.Components {
		// Check for privileged containers in manifests
		for _, manifest := range component.Manifests {
			for _, file := range manifest.Files {
				if err := v.checkManifestSecurity(filepath.Join(packagePath, file), result, component.Name); err != nil {
					result.Warnings = append(result.Warnings, 
						fmt.Sprintf("Failed to analyze manifest security for %s: %v", file, err))
				}
			}
		}
		
		// Check for scripts that might contain secrets
		if len(component.Scripts.Prepare) > 0 || len(component.Scripts.Before) > 0 || len(component.Scripts.After) > 0 {
			allScripts := append(component.Scripts.Prepare, component.Scripts.Before...)
			allScripts = append(allScripts, component.Scripts.After...)
			
			for _, script := range allScripts {
				if containsPotentialSecrets(script) {
					result.Warnings = append(result.Warnings, 
						fmt.Sprintf("Component '%s' script may contain hardcoded secrets or sensitive data", component.Name))
				}
			}
		}
		
		// Check for images from untrusted registries
		for _, image := range component.Images {
			if isUntrustedRegistry(image) {
				result.Warnings = append(result.Warnings, 
					fmt.Sprintf("Component '%s' uses image from potentially untrusted registry: %s", component.Name, image))
			}
		}
	}
	
	return nil
}

// validateResourceConstraints checks for resource management best practices
func (v *PackageValidator) validateResourceConstraints(packagePath string, result *ValidationResult) error {
	zarfYaml, err := util.ReadZarfYaml(filepath.Join(packagePath, "zarf.yaml"))
	if err != nil {
		return fmt.Errorf("failed to read zarf.yaml for resource validation: %w", err)
	}
	
	for _, component := range zarfYaml.Components {
		// Check for large file transfers
		for _, file := range component.Files {
			filePath := filepath.Join(packagePath, file.Source)
			if stat, err := os.Stat(filePath); err == nil {
				sizeInMB := stat.Size() / (1024 * 1024)
				if sizeInMB > 100 { // Files larger than 100MB
					result.Warnings = append(result.Warnings, 
						fmt.Sprintf("Component '%s' includes large file (%dMB): %s", component.Name, sizeInMB, file.Source))
				}
			}
		}
		
		// Check for excessive number of images
		if len(component.Images) > 10 {
			result.Warnings = append(result.Warnings, 
				fmt.Sprintf("Component '%s' includes many images (%d) which may impact package size", component.Name, len(component.Images)))
		}
		
		// Check for charts without resource limits
		for _, chart := range component.Charts {
			// Look for values files that might specify resource limits
			hasResourceLimits := false
			for _, valuesFile := range chart.ValuesFiles {
				valuesPath := filepath.Join(packagePath, valuesFile)
				if content, err := os.ReadFile(valuesPath); err == nil {
					if strings.Contains(string(content), "limits:") || strings.Contains(string(content), "requests:") {
						hasResourceLimits = true
						break
					}
				}
			}
			
			if !hasResourceLimits {
				result.Warnings = append(result.Warnings, 
					fmt.Sprintf("Chart '%s' in component '%s' may not specify resource limits", chart.Name, component.Name))
			}
		}
	}
	
	return nil
}

// Helper functions

// isValidComponentName checks if component name follows conventions
func isValidComponentName(name string) bool {
	// Component names should be lowercase, use hyphens, no spaces
	if strings.Contains(name, " ") {
		return false
	}
	if strings.ToLower(name) != name {
		return false
	}
	return true
}

// hasDependencyCycle detects circular dependencies using DFS
func hasDependencyCycle(start, current string, componentMap map[string]*util.ZarfComponent, visited map[string]bool) bool {
	if current == start && len(visited) > 0 {
		return true
	}
	
	if visited[current] {
		return false
	}
	
	visited[current] = true
	
	if component, exists := componentMap[current]; exists {
		for _, dep := range component.DepsWith {
			if hasDependencyCycle(start, dep, componentMap, visited) {
				return true
			}
		}
	}
	
	return false
}

// checkManifestSecurity analyzes Kubernetes manifests for security issues
func (v *PackageValidator) checkManifestSecurity(manifestPath string, result *ValidationResult, componentName string) error {
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	
	contentStr := string(content)
	
	// Check for privileged security contexts
	if strings.Contains(contentStr, "privileged: true") {
		result.Warnings = append(result.Warnings, 
			fmt.Sprintf("Component '%s' manifest may use privileged containers", componentName))
	}
	
	// Check for host network usage
	if strings.Contains(contentStr, "hostNetwork: true") {
		result.Warnings = append(result.Warnings, 
			fmt.Sprintf("Component '%s' manifest uses host networking", componentName))
	}
	
	// Check for host PID/IPC
	if strings.Contains(contentStr, "hostPID: true") || strings.Contains(contentStr, "hostIPC: true") {
		result.Warnings = append(result.Warnings, 
			fmt.Sprintf("Component '%s' manifest uses host PID or IPC", componentName))
	}
	
	return nil
}

// containsPotentialSecrets checks scripts for potential hardcoded secrets
func containsPotentialSecrets(script string) bool {
	secretPatterns := []string{
		"password=", "PASSWORD=", "token=", "TOKEN=", "secret=", "SECRET=",
		"key=", "KEY=", "api_key", "API_KEY", "aws_access", "AWS_ACCESS",
	}
	
	scriptLower := strings.ToLower(script)
	for _, pattern := range secretPatterns {
		if strings.Contains(scriptLower, strings.ToLower(pattern)) {
			return true
		}
	}
	
	return false
}

// isUntrustedRegistry checks if an image comes from a potentially untrusted registry
func isUntrustedRegistry(image string) bool {
	untrustedRegistries := []string{
		"docker.io/",  // Public Docker Hub (may be OK but worth flagging)
		"index.docker.io/",
	}
	
	// If no registry specified, it defaults to docker.io
	if !strings.Contains(image, "/") || (!strings.Contains(image, ".") && strings.Count(image, "/") == 1) {
		return true
	}
	
	for _, registry := range untrustedRegistries {
		if strings.HasPrefix(image, registry) {
			return true
		}
	}
	
	return false
}
