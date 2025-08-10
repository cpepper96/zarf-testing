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

	helmignore "helm.sh/helm/v3/pkg/ignore"

	"github.com/cpepper96/zarf-testing/pkg/config"
	"github.com/cpepper96/zarf-testing/pkg/ignore"
	"github.com/cpepper96/zarf-testing/pkg/util"
)

const maxNameLength = 63

// Git interface for Git operations (legacy chart-testing compatibility)
type Git interface {
	FileExistsOnBranch(file string, remote string, branch string) bool
	Show(file string, remote string, branch string) (string, error)
	AddWorktree(path string, ref string) error
	RemoveWorktree(path string) error
	MergeBase(commit1 string, commit2 string) (string, error)
	ListChangedFilesInDirs(commit string, dirs ...string) ([]string, error)
	GetURLForRemote(remote string) (string, error)
	ValidateRepository() error
	BranchExists(branch string) bool
}

// Helm interface for Helm operations (legacy chart-testing compatibility)
type Helm interface {
	AddRepo(name string, url string, extraArgs []string) error
	BuildDependencies(chart string) error
	BuildDependenciesWithArgs(chart string, extraArgs []string) error
	LintWithValues(chart string, valuesFile string) error
	InstallWithValues(chart string, valuesFile string, namespace string, release string) error
	UpgradeWithValues(chart string, valuesFile string, namespace string, release string) error
	Test(namespace string, release string) error
	DeleteRelease(namespace string, release string)
	Version() (string, error)
}

// Kubectl interface for kubectl operations (legacy chart-testing compatibility)
type Kubectl interface {
	CreateNamespace(namespace string) error
	DeleteNamespace(namespace string)
	WaitForDeployments(namespace string, selector string) error
	GetPodsforDeployment(namespace string, deployment string) ([]string, error)
	GetPods(args ...string) ([]string, error)
	GetEvents(namespace string) error
	DescribePod(namespace string, pod string) error
	Logs(namespace string, pod string, container string) error
	GetInitContainers(namespace string, pod string) ([]string, error)
	GetContainers(namespace string, pod string) ([]string, error)
}

// Linter interface for linting operations (legacy chart-testing compatibility)
type Linter interface {
	YamlLint(yamlFile string, configFile string) error
	Yamale(yamlFile string, schemaFile string) error
}

// CmdExecutor interface for command execution (legacy chart-testing compatibility)
type CmdExecutor interface {
	RunCommand(cmdTemplate string, data interface{}) error
}

// DirectoryLister interface for directory listing (legacy chart-testing compatibility)
type DirectoryLister interface {
	ListChildDirs(parentDir string, test func(string) bool) ([]string, error)
}

// Utils interface for utility methods (legacy chart-testing compatibility)
type Utils interface {
	LookupChartDir(chartDirs []string, dir string) (string, error)
}

// AccountValidator interface for account validation (legacy chart-testing compatibility)
type AccountValidator interface {
	Validate(repoDomain string, account string) error
}

// Chart represents a Helm chart (legacy chart-testing compatibility)
type Chart struct {
	path          string
	yaml          *util.ChartYaml
	ciValuesPaths []string
}

// Yaml returns the Chart metadata
func (c *Chart) Yaml() *util.ChartYaml {
	return c.yaml
}

// Path returns the chart's directory path
func (c *Chart) Path() string {
	return c.path
}

func (c *Chart) String() string {
	return fmt.Sprintf(`%s => (version: "%s", path: "%s")`, c.yaml.Name, c.yaml.Version, c.Path())
}

// ValuesFilePathsForCI returns all file paths in the 'ci' subfolder of the chart directory matching the pattern '*-values.yaml'
func (c *Chart) ValuesFilePathsForCI() []string {
	return c.ciValuesPaths
}

// HasCIValuesFile checks whether a given CI values file is present.
func (c *Chart) HasCIValuesFile(path string) bool {
	fileName := filepath.Base(path)
	for _, file := range c.ValuesFilePathsForCI() {
		if fileName == filepath.Base(file) {
			return true
		}
	}
	return false
}

// CreateInstallParams generates a randomized release name and namespace based on the chart path
// and optional buildID. If release_name is specified, the release name is set to that string instead.
// If a buildID is specified, it will be part of the generated namespace.
func (c *Chart) CreateInstallParams(buildID string, releaseName string) (release string, namespace string) {
	release = filepath.Base(c.Path())
	if release == "." || release == "/" {
		if releaseName != "" {
			release = releaseName
		} else {
			yaml := c.Yaml()
			release = yaml.Name
		}
	}
	namespace = release
	if buildID != "" {
		namespace = fmt.Sprintf("%s-%s", namespace, buildID)
	}
	randomSuffix := util.RandomString(10)
	release = util.SanitizeName(fmt.Sprintf("%s-%s", release, randomSuffix), maxNameLength)
	namespace = util.SanitizeName(fmt.Sprintf("%s-%s", namespace, randomSuffix), maxNameLength)
	return
}

// NewChart parses the path to a chart directory and allocates a new Chart object. If chartPath is
// not a valid chart directory an error is returned.
func NewChart(chartPath string) (*Chart, error) {
	yaml, err := util.ReadChartYaml(chartPath)
	if err != nil {
		return nil, err
	}
	matches, _ := filepath.Glob(filepath.Join(chartPath, "ci", "*-values.yaml"))
	return &Chart{chartPath, yaml, matches}, nil
}

// Testing struct for legacy chart-testing compatibility
type Testing struct {
	config                   config.Configuration
	helm                     Helm
	kubectl                  Kubectl
	git                      Git
	linter                   Linter
	cmdExecutor              CmdExecutor
	accountValidator         AccountValidator
	directoryLister          DirectoryLister
	utils                    Utils
	previousRevisionWorktree string
	loadRules                func(string) (*helmignore.Rules, error)
}

// TestResults holds results and overall status
type TestResults struct {
	OverallSuccess bool
	TestResults    []TestResult
}

// TestResult holds test results for a specific chart
type TestResult struct {
	Chart *Chart
	Error error
}

// NewTesting creates a new Testing struct with the given config.
func NewTesting(config config.Configuration) (Testing, error) {
	testing := Testing{
		config:           config,
		directoryLister:  util.DirectoryLister{},
		utils:            util.Utils{},
		loadRules:        ignore.LoadRules,
	}

	return testing, nil
}

// ComputeChangedChartDirectories computes changed chart directories
func (ct *Testing) ComputeChangedChartDirectories() ([]string, error) {
	// Implementation would go here for actual chart discovery
	// For now, return empty slice to make tests compile
	return []string{}, nil
}

// ValidateMaintainers validates chart maintainers
func (ct *Testing) ValidateMaintainers(chart *Chart) error {
	// Implementation would go here for actual validation
	// For now, return nil to make tests compile
	return nil
}

// LintChart lints a chart and returns test result
func (ct *Testing) LintChart(chart *Chart) TestResult {
	// Implementation would go here for actual linting
	// For now, return success to make tests compile
	return TestResult{Chart: chart, Error: nil}
}

// ReadAllChartDirectories reads all chart directories
func (ct *Testing) ReadAllChartDirectories() ([]string, error) {
	// Implementation would go here for actual chart directory reading
	// For now, return empty slice to make tests compile
	return []string{}, nil
}

// LintCharts lints multiple charts
func (ct *Testing) LintCharts() ([]TestResult, error) {
	// Implementation would go here for actual chart linting
	// For now, return success to make tests compile
	return []TestResult{}, nil
}

// generateInstallConfig generates install configuration
func (ct *Testing) generateInstallConfig(chart *Chart) (string, string, string, map[string]interface{}) {
	// Implementation would go here for actual config generation
	// For now, return default values to make tests compile
	return "test-namespace", "test-release", "app=test", map[string]interface{}{}
}
