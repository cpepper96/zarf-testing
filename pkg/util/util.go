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

package util

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/hashicorp/go-multierror"
	"gopkg.in/yaml.v2"
)

const chars = "1234567890abcdefghijklmnopqrstuvwxyz"

type Maintainer struct {
	Name  string `yaml:"name"`
	Email string `yaml:"email"`
}

type ChartYaml struct {
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	Deprecated  bool   `yaml:"deprecated"`
	Maintainers []Maintainer
}

type ZarfYaml struct {
	Kind     string `yaml:"kind"`
	Metadata struct {
		Name         string `yaml:"name"`
		Description  string `yaml:"description"`
		Version      string `yaml:"version"`
		Architecture string `yaml:"architecture,omitempty"`
		Deprecated   bool   `yaml:"deprecated,omitempty"`
	} `yaml:"metadata"`
	Variables []ZarfVariable  `yaml:"variables,omitempty"`
	Constants []ZarfConstant  `yaml:"constants,omitempty"`
	Components []ZarfComponent `yaml:"components,omitempty"`
}

type ZarfVariable struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	Default     string `yaml:"default,omitempty"`
	Prompt      bool   `yaml:"prompt,omitempty"`
}

type ZarfConstant struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

type ZarfComponent struct {
	Name        string              `yaml:"name"`
	Description string              `yaml:"description,omitempty"`
	Default     bool                `yaml:"default,omitempty"`
	Required    bool                `yaml:"required,omitempty"`
	Only        ZarfComponentOnly   `yaml:"only,omitempty"`
	Group       string              `yaml:"group,omitempty"`
	DepsWith    []string            `yaml:"depsWith,omitempty"`
	Files       []ZarfFile          `yaml:"files,omitempty"`
	Charts      []ZarfChart         `yaml:"charts,omitempty"`
	Manifests   []ZarfManifest      `yaml:"manifests,omitempty"`
	Images      []string            `yaml:"images,omitempty"`
	Repos       []string            `yaml:"repos,omitempty"`
	DataInjections []ZarfDataInjection `yaml:"dataInjections,omitempty"`
	Scripts     ZarfComponentScripts `yaml:"scripts,omitempty"`
}

type ZarfComponentOnly struct {
	LocalOS      string `yaml:"localOS,omitempty"`
	Cluster      ZarfComponentOnlyCluster `yaml:"cluster,omitempty"`
}

type ZarfComponentOnlyCluster struct {
	Architecture string   `yaml:"architecture,omitempty"`
	Distros      []string `yaml:"distros,omitempty"`
}

type ZarfFile struct {
	Source      string   `yaml:"source"`
	Target      string   `yaml:"target"`
	Shasum      string   `yaml:"shasum,omitempty"`
	Executable  bool     `yaml:"executable,omitempty"`
	ExtractPath string   `yaml:"extractPath,omitempty"`
}

type ZarfChart struct {
	Name         string                 `yaml:"name"`
	Version      string                 `yaml:"version,omitempty"`
	Url          string                 `yaml:"url,omitempty"`
	RepoName     string                 `yaml:"repoName,omitempty"`
	GitPath      string                 `yaml:"gitPath,omitempty"`
	LocalPath    string                 `yaml:"localPath,omitempty"`
	Namespace    string                 `yaml:"namespace,omitempty"`
	ReleaseName  string                 `yaml:"releaseName,omitempty"`
	NoWait       bool                   `yaml:"noWait,omitempty"`
	ValuesFiles  []string               `yaml:"valuesFiles,omitempty"`
	Variables    []ZarfChartVariable    `yaml:"variables,omitempty"`
}

type ZarfChartVariable struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	Path        string `yaml:"path"`
}

type ZarfManifest struct {
	Name              string   `yaml:"name"`
	Namespace         string   `yaml:"namespace,omitempty"`
	Files             []string `yaml:"files,omitempty"`
	KustomizeAllowAnyOf []string `yaml:"kustomizeAllowAnyOf,omitempty"`
	Kustomizations    []string `yaml:"kustomizations,omitempty"`
}

type ZarfDataInjection struct {
	Source   string `yaml:"source"`
	Target   ZarfContainerTarget `yaml:"target"`
	Compress bool   `yaml:"compress,omitempty"`
}

type ZarfContainerTarget struct {
	Namespace string `yaml:"namespace"`
	Selector  string `yaml:"selector"`
	Container string `yaml:"container"`
	Path      string `yaml:"path"`
}

type ZarfComponentScripts struct {
	ShowOutput bool     `yaml:"showOutput,omitempty"`
	TimeoutSeconds int  `yaml:"timeoutSeconds,omitempty"`
	Retry      bool     `yaml:"retry,omitempty"`
	Prepare    []string `yaml:"prepare,omitempty"`
	Before     []string `yaml:"before,omitempty"`
	After      []string `yaml:"after,omitempty"`
}

func Flatten(items []interface{}) ([]string, error) {
	return doFlatten([]string{}, items)
}

func init() {
	rand.New(rand.NewSource(time.Now().UnixNano())) // nolint: gosec
}

func doFlatten(result []string, items interface{}) ([]string, error) {
	var err error

	switch v := items.(type) {
	case string:
		result = append(result, v)
	case []string:
		result = append(result, v...)
	case []interface{}:
		for _, item := range v {
			result, err = doFlatten(result, item)
			if err != nil {
				return nil, err
			}
		}
	default:
		return nil, fmt.Errorf("flatten does not support %T", v)
	}

	return result, err
}

func StringSliceContains(slice []string, s string) bool {
	for _, element := range slice {
		if s == element {
			return true
		}
	}
	return false
}

func FileExists(file string) bool {
	if _, err := os.Stat(file); err != nil {
		return false
	}
	return true
}

// RandomString string creates a random string of numbers and lower-case ascii characters with the specified length.
func RandomString(length int) string {
	n := len(chars)
	bytes := make([]byte, length)
	for i := range bytes {
		bytes[i] = chars[rand.Intn(n)] // nolint: gosec
	}
	return string(bytes)
}

type DirectoryLister struct{}

// ListChildDirs lists subdirectories of parentDir matching the test function.
func (l DirectoryLister) ListChildDirs(parentDir string, test func(dir string) bool) ([]string, error) {
	entries, err := os.ReadDir(parentDir)
	if err != nil {
		return nil, err
	}
	fileInfos := make([]fs.FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		fileInfos = append(fileInfos, info)
	}

	var dirs []string
	for _, dir := range fileInfos {
		dirName := dir.Name()
		parentSlashChildDir := filepath.Join(parentDir, dirName)
		if test(parentSlashChildDir) {
			dirs = append(dirs, parentSlashChildDir)
		}
	}

	return dirs, nil
}

type Utils struct{}

func (u Utils) LookupChartDir(chartDirs []string, dir string) (string, error) {
	for _, chartDir := range chartDirs {
		currentDir := dir
		for {
			chartYaml := filepath.Join(currentDir, "Chart.yaml")
			parent := filepath.Dir(filepath.Dir(chartYaml))
			chartDir = strings.TrimRight(chartDir, "/") // remove any trailing slash from the dir

			// check directory has a Chart.yaml and that it is in a
			// direct subdirectory of a configured charts directory
			if FileExists(chartYaml) && (parent == chartDir) {
				return currentDir, nil
			}

			currentDir = filepath.Dir(currentDir)
			relativeDir, _ := filepath.Rel(chartDir, currentDir)
			joined := filepath.Join(chartDir, relativeDir)
			if (joined == chartDir) || strings.HasPrefix(relativeDir, "..") {
				break
			}
		}
	}
	return "", errors.New("no chart directory")
}

// ReadChartYaml attempts to parse Chart.yaml within the specified directory
// and return a newly allocated ChartYaml object. If no Chart.yaml is present
// or there is an error unmarshaling the file contents, an error will be returned.
func ReadChartYaml(dir string) (*ChartYaml, error) {
	yamlBytes, err := os.ReadFile(filepath.Join(dir, "Chart.yaml"))
	if err != nil {
		return nil, fmt.Errorf("could not read 'Chart.yaml': %w", err)
	}
	return UnmarshalChartYaml(yamlBytes)
}

// UnmarshalChartYaml parses the yaml encoded data and returns a newly
// allocated ChartYaml object.
func UnmarshalChartYaml(yamlBytes []byte) (*ChartYaml, error) {
	chartYaml := &ChartYaml{}
	if err := yaml.Unmarshal(yamlBytes, chartYaml); err != nil {
		return nil, fmt.Errorf("could not unmarshal 'Chart.yaml': %w", err)
	}
	return chartYaml, nil
}

// ReadZarfYaml attempts to parse zarf.yaml within the specified directory
// and return a newly allocated ZarfYaml object. If no zarf.yaml is present
// or there is an error unmarshaling the file contents, an error will be returned.
func ReadZarfYaml(path string) (*ZarfYaml, error) {
	yamlBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read 'zarf.yaml': %w", err)
	}
	return UnmarshalZarfYaml(yamlBytes)
}

// UnmarshalZarfYaml parses the yaml encoded data and returns a newly
// allocated ZarfYaml object.
func UnmarshalZarfYaml(yamlBytes []byte) (*ZarfYaml, error) {
	zarfYaml := &ZarfYaml{}
	if err := yaml.Unmarshal(yamlBytes, zarfYaml); err != nil {
		return nil, fmt.Errorf("could not unmarshal 'zarf.yaml': %w", err)
	}
	return zarfYaml, nil
}

func CompareVersions(left string, right string) (int, error) {
	leftVersion, err := semver.NewVersion(left)
	if err != nil {
		return 0, fmt.Errorf("failed parsing semantic version: %w", err)
	}
	rightVersion, err := semver.NewVersion(right)
	if err != nil {
		return 0, fmt.Errorf("failed parsing semantic version: %w", err)
	}
	return leftVersion.Compare(rightVersion), nil
}

func BreakingChangeAllowed(left string, right string) (bool, error) {
	leftVersion, err := semver.NewVersion(left)
	if err != nil {
		return false, fmt.Errorf("failed parsing semantic version: %w", err)
	}
	rightVersion, err := semver.NewVersion(right)
	if err != nil {
		return false, fmt.Errorf("failed parsing semantic version: %w", err)
	}

	constraintOp := "^"
	if leftVersion.Major() == 0 {
		constraintOp = "~"
	}
	c, err := semver.NewConstraint(fmt.Sprintf("%s %s", constraintOp, leftVersion.String()))
	if err != nil {
		return false, fmt.Errorf("failed parsing semantic version constraint: %w", err)
	}

	minor, reasons := c.Validate(rightVersion)
	if len(reasons) > 0 {
		err = multierror.Append(err, reasons...)
	}

	return !minor, err
}

func PrintDelimiterLineToWriter(w io.Writer, delimiterChar string) {
	fmt.Fprintln(w, strings.Repeat(delimiterChar, 120))
}

func GithubGroupsBegin(w io.Writer, title string) {
	fmt.Fprintf(w, "::group::%s\n", title)
}

func GithubGroupsEnd(w io.Writer) {
	fmt.Fprintln(w, "::endgroup::")
}

func SanitizeName(s string, maxLength int) string {
	reg := regexp.MustCompile("^[^a-zA-Z0-9]+")

	excess := len(s) - maxLength
	result := s
	if excess > 0 {
		result = s[excess:]
	}
	return reg.ReplaceAllString(result, "")
}

func GetRandomPort() (int, error) {
	listener, err := net.Listen("tcp", ":0") // nolint: gosec
	defer listener.Close()                   // nolint: staticcheck
	if err != nil {
		return 0, err
	}

	return listener.Addr().(*net.TCPAddr).Port, nil
}
