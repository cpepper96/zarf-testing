# Zarf-Testing Agent Guide

## Project Overview

**Zarf-Testing** is a tool for testing Zarf packages, adapted from helm/chart-testing. It provides functionality to lint, validate, and test Zarf packages in CI/CD pipelines, with particular focus on airgapped and disconnected environments.

## Project Structure

```
/
├── zt/                     # Main CLI application (renamed from ct/)
│   ├── main.go            # Entry point
│   └── cmd/               # Cobra CLI commands
├── pkg/                   # Core packages
│   ├── zarf/             # Zarf package handling (renamed from chart/)
│   ├── config/           # Configuration management
│   ├── tool/             # External tool integrations
│   ├── util/             # Utilities
│   ├── exec/             # Command execution
│   └── ignore/           # Ignore file handling
├── etc/                  # Configuration files and schemas
├── examples/             # Usage examples
├── doc/                  # Documentation
└── tests/                # Test suites
```

## Build, Test, and Development Commands

### Build Commands
```bash
# Build the binary
go build ./zt

# Build with specific output
go build -o zt ./zt

# Cross-platform build using Goreleaser
./build.sh

# Release build
./build.sh --release
```

### Test Commands
```bash
# Run unit tests
go test ./...

# Run tests with coverage
go test -race -coverprofile=coverage.out ./...

# Run specific package tests
go test ./pkg/zarf/...

# Integration tests
go test -tags=integration ./tests/integration/...

# E2E tests (requires Kubernetes cluster)
go test -tags=e2e ./tests/e2e/...
```

### Development Commands
```bash
# Install dependencies
go mod tidy

# Lint code
golangci-lint run

# Format code
gofmt -s -w .

# Generate documentation
go run ./zt doc-gen --output-dir ./doc

# Run development version
go run ./zt --help
```

### Quality Checks
```bash
# Static analysis
go vet ./...

# Security scanning
gosec ./...

# Dependency vulnerability check
govulncheck ./...
```

## Code Style and Conventions

### Go Code Style
- Follow standard Go conventions (gofmt, go vet)
- Use meaningful variable and function names
- Prefer short, descriptive names for local variables
- Use interfaces for testability and modularity
- Follow error handling patterns: `if err != nil { return err }`

### Package Structure
- **cmd/**: CLI command implementations using Cobra
- **pkg/**: Reusable packages, importable by other projects
- **internal/**: Private application code
- **tool/**: External tool integrations (Zarf SDK, Git, Kubectl)

### Naming Conventions
- Use `zarf` prefix for Zarf-specific concepts (zarfPackage, zarfConfig)
- Keep chart-testing compatibility where possible (config field names)
- CLI commands: kebab-case (`list-changed`, `lint-and-deploy`)
- Go identifiers: camelCase/PascalCase per Go conventions

### Error Handling
- Return errors, don't panic in normal operation
- Wrap errors with context: `fmt.Errorf("failed to validate package %s: %w", path, err)`
- Use structured logging for debug information
- Provide user-friendly error messages

## Architecture and Design Patterns

### Core Architecture
```
CLI Layer (zt/cmd/) 
    ↓
Business Logic (pkg/zarf/)
    ↓  
Tool Integration (pkg/tool/)
    ↓
External Tools (Zarf SDK, Git, Kubectl)
```

### Key Patterns
- **Command Pattern**: Each CLI command encapsulates operation logic
- **Interface Segregation**: Small, focused interfaces for tool integrations
- **Dependency Injection**: Inject tool interfaces for testing
- **Factory Pattern**: Create configured instances based on user config

### Critical Interfaces
```go
// Core testing interface
type PackageTesting interface {
    Lint(packages []string) error
    Deploy(packages []string) error
    ListChanged() ([]string, error)
}

// Tool interfaces
type ZarfClient interface {
    ValidatePackage(path string) (*ValidationResult, error)
    DeployPackage(path string, config *DeployConfig) error
}

type GitClient interface {
    FindChangedFiles(remote, target string) ([]string, error)
    GetCurrentBranch() (string, error)
}
```

### Configuration System
- Uses Viper for hierarchical configuration (CLI > ENV > Config File)
- Supports YAML, JSON, TOML configuration formats
- Environment variables prefixed with `ZT_`
- Default config search paths: `.`, `$HOME/.zt`, `/etc/zt`

## Testing Guidelines

### Unit Tests
- Test all public functions and methods
- Use table-driven tests for multiple scenarios
- Mock external dependencies using interfaces
- Target >80% code coverage
- Place tests in `*_test.go` files alongside source

### Integration Tests
- Test complete workflows end-to-end
- Use real Zarf packages in `tests/fixtures/`
- Test configuration loading and CLI integration
- Run in CI with docker containers

### E2E Tests
- Test against real Kubernetes clusters
- Use Kind clusters for CI testing
- Test actual Zarf package deployment
- Clean up resources after tests

### Test Structure
```
tests/
├── unit/           # Unit tests
├── integration/    # Integration tests
├── e2e/           # End-to-end tests
├── fixtures/      # Test data and Zarf packages
└── mocks/         # Generated mocks
```

## Security Considerations

### Secrets and Sensitive Data
- Never log secrets or authentication tokens
- Use secure credential handling for private registries
- Sanitize command output that might contain sensitive data
- Support credential injection via environment variables

### Package Validation
- Validate Zarf package structure and content
- Check for suspicious files or configurations
- Verify package signatures when available
- Scan for known vulnerabilities in included images

### Execution Safety
- Validate user input for command injection
- Use structured command execution (exec.Command)
- Limit file system access to necessary directories
- Run with minimal required permissions

## Zarf-Specific Implementation Notes

### Zarf SDK Integration
- Primary challenge: SDK initialization and usage patterns
- Use Zarf SDK for package validation and deployment
- Fallback to Zarf CLI wrapper if SDK integration fails
- Document working patterns and limitations

### Package Discovery
- Look for `zarf.yaml` files in specified directories
- Support multiple package directories
- Handle Zarf package validation and structure checking
- Git-based change detection for modified packages

### Airgap Considerations
- Support for offline/disconnected environments
- Local package registry integration
- Image and artifact bundling validation
- Network-independent testing capabilities

## Migration from Chart-Testing

### Compatibility Layer
- Maintain similar configuration structure where possible
- Support chart-testing-style CI/CD integration
- Preserve familiar command patterns and output formats
- Provide migration guide and examples

### Key Differences
- `zarf.yaml` instead of `Chart.yaml`
- Zarf SDK instead of Helm SDK
- Package deployment instead of chart installation
- Component-based instead of template-based validation

## Development Workflow

### Feature Development
1. Create feature branch from main
2. Implement with unit tests
3. Add integration tests if needed
4. Update documentation
5. Submit PR with comprehensive testing

### Critical Path Dependencies
- Zarf SDK stability and documentation
- Kubernetes cluster access for testing
- Git integration for change detection
- Chart-testing architecture compatibility

### Performance Considerations
- Efficient package discovery in large repositories
- Parallel processing of multiple packages
- Memory usage with large Zarf packages
- Network efficiency in airgapped scenarios
