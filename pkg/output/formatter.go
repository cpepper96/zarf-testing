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

package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
)

// Format represents the output format type
type Format int

const (
	// FormatText represents plain text output
	FormatText Format = iota
	// FormatJSON represents JSON output
	FormatJSON
	// FormatGitHub represents GitHub Actions compatible output
	FormatGitHub
)

// Config contains output formatting configuration
type Config struct {
	Format      Format
	NoColor     bool
	GithubGroups bool
	Writer      io.Writer
}

// Formatter handles output formatting with colors and different formats
type Formatter struct {
	config     *Config
	jsonBuffer []interface{}
}

// NewFormatter creates a new output formatter
func NewFormatter(config *Config) *Formatter {
	if config.Writer == nil {
		config.Writer = os.Stdout
	}
	
	// Disable colors if requested or if not a terminal
	if config.NoColor {
		color.NoColor = true
	}
	
	return &Formatter{
		config:     config,
		jsonBuffer: make([]interface{}, 0),
	}
}

// Success prints a success message
func (f *Formatter) Success(msg string, args ...interface{}) {
	message := fmt.Sprintf(msg, args...)
	
	switch f.config.Format {
	case FormatJSON:
		f.addJSONEvent("success", message, nil)
	case FormatGitHub:
		if f.config.GithubGroups {
			fmt.Fprintf(f.config.Writer, "✅ %s\n", message)
		} else {
			fmt.Fprintf(f.config.Writer, "✅ %s\n", message)
		}
	default:
		green := color.New(color.FgGreen, color.Bold)
		fmt.Fprintf(f.config.Writer, "%s %s\n", green.Sprint("✅"), message)
	}
}

// Error prints an error message
func (f *Formatter) Error(msg string, args ...interface{}) {
	message := fmt.Sprintf(msg, args...)
	
	switch f.config.Format {
	case FormatJSON:
		f.addJSONEvent("error", message, nil)
	case FormatGitHub:
		fmt.Fprintf(f.config.Writer, "❌ %s\n", message)
	default:
		red := color.New(color.FgRed, color.Bold)
		fmt.Fprintf(f.config.Writer, "%s %s\n", red.Sprint("❌"), message)
	}
}

// Warning prints a warning message
func (f *Formatter) Warning(msg string, args ...interface{}) {
	message := fmt.Sprintf(msg, args...)
	
	switch f.config.Format {
	case FormatJSON:
		f.addJSONEvent("warning", message, nil)
	case FormatGitHub:
		fmt.Fprintf(f.config.Writer, "⚠️  %s\n", message)
	default:
		yellow := color.New(color.FgYellow, color.Bold)
		fmt.Fprintf(f.config.Writer, "%s %s\n", yellow.Sprint("⚠️"), message)
	}
}

// Info prints an informational message
func (f *Formatter) Info(msg string, args ...interface{}) {
	message := fmt.Sprintf(msg, args...)
	
	switch f.config.Format {
	case FormatJSON:
		f.addJSONEvent("info", message, nil)
	case FormatGitHub:
		fmt.Fprintf(f.config.Writer, "ℹ️  %s\n", message)
	default:
		blue := color.New(color.FgBlue)
		fmt.Fprintf(f.config.Writer, "%s %s\n", blue.Sprint("ℹ️"), message)
	}
}

// Progress prints a progress message
func (f *Formatter) Progress(msg string, args ...interface{}) {
	message := fmt.Sprintf(msg, args...)
	
	switch f.config.Format {
	case FormatJSON:
		f.addJSONEvent("progress", message, nil)
	case FormatGitHub:
		fmt.Fprintf(f.config.Writer, "🔧 %s\n", message)
	default:
		cyan := color.New(color.FgCyan)
		fmt.Fprintf(f.config.Writer, "%s %s\n", cyan.Sprint("🔧"), message)
	}
}

// Section prints a section header
func (f *Formatter) Section(title string) {
	switch f.config.Format {
	case FormatJSON:
		f.addJSONEvent("section", title, nil)
	case FormatGitHub:
		if f.config.GithubGroups {
			fmt.Fprintf(f.config.Writer, "::group::%s\n", title)
		} else {
			fmt.Fprintf(f.config.Writer, "\n📋 %s\n", title)
		}
	default:
		bold := color.New(color.Bold, color.FgMagenta)
		fmt.Fprintf(f.config.Writer, "\n%s %s\n", bold.Sprint("📋"), bold.Sprint(title))
	}
}

// EndSection ends a section (mainly for GitHub Actions groups)
func (f *Formatter) EndSection() {
	if f.config.Format == FormatGitHub && f.config.GithubGroups {
		fmt.Fprintf(f.config.Writer, "::endgroup::\n")
	}
}

// Step prints a step within a section
func (f *Formatter) Step(current, total int, msg string, args ...interface{}) {
	message := fmt.Sprintf(msg, args...)
	
	switch f.config.Format {
	case FormatJSON:
		data := map[string]interface{}{
			"current": current,
			"total":   total,
		}
		f.addJSONEvent("step", message, data)
	case FormatGitHub:
		fmt.Fprintf(f.config.Writer, "  [%d/%d] %s\n", current, total, message)
	default:
		cyan := color.New(color.FgCyan)
		fmt.Fprintf(f.config.Writer, "  %s [%d/%d] %s\n", cyan.Sprint("→"), current, total, message)
	}
}

// PrintJSON outputs all buffered events as JSON
func (f *Formatter) PrintJSON() error {
	if f.config.Format != FormatJSON {
		return nil
	}
	
	output := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"events":    f.jsonBuffer,
	}
	
	encoder := json.NewEncoder(f.config.Writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// addJSONEvent adds an event to the JSON buffer
func (f *Formatter) addJSONEvent(eventType, message string, data map[string]interface{}) {
	event := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"type":      eventType,
		"message":   message,
	}
	
	if data != nil {
		event["data"] = data
	}
	
	f.jsonBuffer = append(f.jsonBuffer, event)
}

// ProgressBar creates a simple progress bar
type ProgressBar struct {
	formatter *Formatter
	total     int
	current   int
	title     string
}

// NewProgressBar creates a new progress bar
func (f *Formatter) NewProgressBar(title string, total int) *ProgressBar {
	return &ProgressBar{
		formatter: f,
		total:     total,
		current:   0,
		title:     title,
	}
}

// Update updates the progress bar
func (pb *ProgressBar) Update(current int, message string) {
	pb.current = current
	
	switch pb.formatter.config.Format {
	case FormatJSON:
		data := map[string]interface{}{
			"current": current,
			"total":   pb.total,
			"percent": float64(current) / float64(pb.total) * 100,
		}
		pb.formatter.addJSONEvent("progress_update", message, data)
	case FormatGitHub:
		percent := float64(current) / float64(pb.total) * 100
		fmt.Fprintf(pb.formatter.config.Writer, "Progress: %.1f%% (%d/%d) - %s\n", percent, current, pb.total, message)
	default:
		percent := float64(current) / float64(pb.total) * 100
		bar := pb.generateBar(50, percent)
		
		cyan := color.New(color.FgCyan)
		fmt.Fprintf(pb.formatter.config.Writer, "\r%s [%s] %.1f%% (%d/%d) %s", 
			cyan.Sprint("🔧"), bar, percent, current, pb.total, message)
		
		if current == pb.total {
			fmt.Fprintln(pb.formatter.config.Writer) // New line when complete
		}
	}
}

// generateBar creates a visual progress bar
func (pb *ProgressBar) generateBar(width int, percent float64) string {
	filled := int(percent / 100 * float64(width))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return bar
}

// Finish completes the progress bar
func (pb *ProgressBar) Finish(message string) {
	pb.Update(pb.total, message)
}
