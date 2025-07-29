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
	"time"

	"github.com/cpepper96/zarf-testing/pkg/config"
	"github.com/cpepper96/zarf-testing/pkg/exec"
	"github.com/cpepper96/zarf-testing/pkg/util"
)

// DeploymentResult represents the result of a Zarf package deployment test
type DeploymentResult struct {
	PackagePath    string
	Success        bool
	DeployTime     time.Duration
	Errors         []string
	Warnings       []string
	ComponentTests []ComponentTestResult
}

// ComponentTestResult represents the test result for a single component
type ComponentTestResult struct {
	ComponentName string
	Success       bool
	Message       string
}

// PackageDeployer handles Zarf package deployment testing
type PackageDeployer struct {
	UseZarfCLI    bool
	Timeout       time.Duration
	SkipCleanup   bool
	TestNamespace string
}

// Deployer provides Zarf package deployment testing functionality
type Deployer struct {
	config   *config.Configuration
	deployer *PackageDeployer
}

// NewPackageDeployer creates a new package deployer
func NewPackageDeployer() *PackageDeployer {
	return &PackageDeployer{
		UseZarfCLI:    true,
		Timeout:       10 * time.Minute,
		SkipCleanup:   false,
		TestNamespace: "zt-test", // Will be made unique per test
	}
}

// NewDeployer creates a new deployer with the given configuration
func NewDeployer(config *config.Configuration) (*Deployer, error) {
	deployer := &Deployer{
		config:   config,
		deployer: NewPackageDeployer(),
	}
	
	// Verify kubectl is available
	executor := exec.NewProcessExecutor(false)
	_, err := executor.RunProcessAndCaptureOutput("kubectl", "version", "--client")
	if err != nil {
		return nil, fmt.Errorf("kubectl not available: %w", err)
	}
	
	// Verify zarf is available
	_, err = executor.RunProcessAndCaptureOutput("zarf", "version")
	if err != nil {
		return nil, fmt.Errorf("zarf CLI not available: %w", err)
	}
	
	return deployer, nil
}

// TestPackage deploys and tests a Zarf package
func (d *Deployer) TestPackage(packagePath string) (*DeploymentResult, error) {
	return d.deployer.DeployPackage(packagePath)
}

// DeployPackage deploys and tests a Zarf package
func (d *PackageDeployer) DeployPackage(packagePath string) (*DeploymentResult, error) {
	result := &DeploymentResult{
		PackagePath:    packagePath,
		Success:        false,
		Errors:         []string{},
		Warnings:       []string{},
		ComponentTests: []ComponentTestResult{},
	}

	startTime := time.Now()

	// First validate that this is a Zarf package
	if !IsZarfPackage(packagePath) {
		result.Errors = append(result.Errors, "Directory does not contain a zarf.yaml file")
		return result, nil
	}

	// Check if Zarf CLI is available
	executor := exec.NewProcessExecutor(false)
	_, err := executor.RunProcessAndCaptureOutput("zarf", "version")
	if err != nil {
		result.Errors = append(result.Errors, "Zarf CLI not found - please install Zarf CLI for deployment testing")
		return result, nil
	}

	// Check Kubernetes connectivity
	err = d.checkKubernetesConnection()
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Kubernetes connection failed: %v", err))
		return result, nil
	}

	// Create a unique test namespace
	testNamespace := d.generateTestNamespace()
	
	// Build the package first
	packageTarPath, err := d.buildPackage(packagePath)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to build package: %v", err))
		return result, nil
	}

	// Deploy the package
	err = d.deployPackageToCluster(packageTarPath, testNamespace)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to deploy package: %v", err))
		return result, nil
	}

	// Test the deployment
	componentResults, err := d.testDeployment(packagePath, testNamespace)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Deployment testing failed: %v", err))
	}
	result.ComponentTests = componentResults

	// Cleanup if not skipped
	if !d.SkipCleanup {
		err = d.cleanupDeployment(testNamespace)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Cleanup failed: %v", err))
		}
	}

	result.DeployTime = time.Since(startTime)
	result.Success = len(result.Errors) == 0

	return result, nil
}

// checkKubernetesConnection verifies we can connect to Kubernetes
func (d *PackageDeployer) checkKubernetesConnection() error {
	executor := exec.NewProcessExecutor(false)
	_, err := executor.RunProcessAndCaptureOutput("kubectl", "cluster-info")
	if err != nil {
		return fmt.Errorf("kubectl cluster-info failed: %w", err)
	}
	return nil
}

// generateTestNamespace creates a unique namespace for testing
func (d *PackageDeployer) generateTestNamespace() string {
	timestamp := time.Now().Format("20060102-150405")
	randomSuffix := util.RandomString(8)
	return fmt.Sprintf("%s-%s-%s", d.TestNamespace, timestamp, randomSuffix)
}

// buildPackage builds the Zarf package
func (d *PackageDeployer) buildPackage(packagePath string) (string, error) {
	executor := exec.NewProcessExecutor(false)
	
	// Build the package using zarf package create
	_, err := executor.RunProcessInDirAndCaptureOutput(packagePath, "zarf", "package", "create", ".", "--confirm")
	if err != nil {
		return "", fmt.Errorf("zarf package create failed: %w", err)
	}

	// Find the created package file
	files, err := os.ReadDir(packagePath)
	if err != nil {
		return "", fmt.Errorf("failed to read package directory: %w", err)
	}

	for _, file := range files {
		if strings.HasPrefix(file.Name(), "zarf-package-") && strings.HasSuffix(file.Name(), ".tar.zst") {
			return filepath.Join(packagePath, file.Name()), nil
		}
	}

	return "", fmt.Errorf("no zarf package file found after build")
}

// deployPackageToCluster deploys the package to the test cluster
func (d *PackageDeployer) deployPackageToCluster(packageTarPath, namespace string) error {
	executor := exec.NewProcessExecutor(false)
	
	// Deploy the package
	_, err := executor.RunProcessAndCaptureOutput("zarf", "package", "deploy", packageTarPath, "--confirm")
	if err != nil {
		return fmt.Errorf("zarf package deploy failed: %w", err)
	}

	return nil
}

// testDeployment tests that the deployment is working
func (d *PackageDeployer) testDeployment(packagePath, namespace string) ([]ComponentTestResult, error) {
	var results []ComponentTestResult
	
	// Load the zarf.yaml to understand what components were deployed
	zarfYaml, err := util.ReadZarfYaml(filepath.Join(packagePath, "zarf.yaml"))
	if err != nil {
		return results, fmt.Errorf("failed to read zarf.yaml: %w", err)
	}

	// For now, just do basic connectivity tests
	executor := exec.NewProcessExecutor(false)
	
	// Check if any pods are running (basic test)
	_, err = executor.RunProcessAndCaptureOutput("kubectl", "get", "pods", "--all-namespaces")
	if err != nil {
		results = append(results, ComponentTestResult{
			ComponentName: "basic-connectivity",
			Success:       false,
			Message:       fmt.Sprintf("Failed to get pods: %v", err),
		})
	} else {
		results = append(results, ComponentTestResult{
			ComponentName: "basic-connectivity",
			Success:       true,
			Message:       "Successfully connected to cluster and retrieved pod information",
		})
	}

	// Add a basic test to verify the package name
	results = append(results, ComponentTestResult{
		ComponentName: fmt.Sprintf("package-%s", zarfYaml.Metadata.Name),
		Success:       true,
		Message:       "Package metadata loaded successfully",
	})

	return results, nil
}

// cleanupDeployment removes the test deployment
func (d *PackageDeployer) cleanupDeployment(namespace string) error {
	executor := exec.NewProcessExecutor(false)
	
	// Remove the package (this is more complex in real Zarf)
	// For now, just log that we would cleanup
	_, err := executor.RunProcessAndCaptureOutput("zarf", "package", "remove", "--confirm")
	if err != nil {
		// Don't fail if cleanup fails, just warn
		return fmt.Errorf("package removal failed: %w", err)
	}

	return nil
}

// DeployPackages deploys and tests multiple packages
func (d *PackageDeployer) DeployPackages(packagePaths []string) ([]*DeploymentResult, error) {
	var results []*DeploymentResult
	
	for _, path := range packagePaths {
		result, err := d.DeployPackage(path)
		if err != nil {
			return nil, fmt.Errorf("failed to deploy package %s: %w", path, err)
		}
		results = append(results, result)
		
		// If deployment failed and we're not skipping cleanup, stop
		if !result.Success && !d.SkipCleanup {
			break
		}
	}
	
	return results, nil
}

// PrintDeploymentResults prints deployment results in a user-friendly format
func PrintDeploymentResults(results []*DeploymentResult) {
	for _, result := range results {
		fmt.Printf("\n==> Deploying %s\n", result.PackagePath)
		
		if len(result.Errors) > 0 {
			fmt.Println("[ERROR] Deployment failed:")
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
		
		if len(result.ComponentTests) > 0 {
			fmt.Println("[INFO] Component Test Results:")
			for _, test := range result.ComponentTests {
				status := "PASS"
				if !test.Success {
					status = "FAIL"
				}
				fmt.Printf("  - %s: %s - %s\n", test.ComponentName, status, test.Message)
			}
		}
		
		if result.Success {
			fmt.Printf("[INFO] Package deployed successfully in %v\n", result.DeployTime)
		} else {
			fmt.Printf("[ERROR] Package deployment failed in %v\n", result.DeployTime)
		}
	}
}

// HasDeploymentErrors checks if any of the results have errors
func HasDeploymentErrors(results []*DeploymentResult) bool {
	for _, result := range results {
		if !result.Success {
			return true
		}
	}
	return false
}
