# Zarf-Testing

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Report Card](https://goreportcard.com/badge/github.com/cpepper96/zarf-testing)](https://goreportcard.com/report/github.com/cpepper96/zarf-testing)

`zt` is the testing tool for [Zarf](https://zarf.dev) packages. It provides comprehensive validation, linting, and deployment testing capabilities for Zarf packages, going beyond what the basic `zarf dev lint` command offers.

Adapted from [helm/chart-testing](https://github.com/helm/chart-testing)

## 🚀 Features
- 🔍 **Advanced Package Linting**: Beyond basic Zarf CLI validation
- 📦 **Package Discovery**: Automatic detection of changed packages via Git
- 🔄 **Version Increment Validation**: Ensures version bumps when packages change
- 🖼️ **Image Pinning Validation**: Enforces container image digest pinning
- 🎨 **Rich Output Formatting**: Colored text, JSON, and GitHub Actions formats
- ⚙️ **Flexible Configuration**: Viper-based config with Zarf-specific options

## 📥 Installation

### Prerequisites

- [Zarf CLI](https://zarf.dev) (for `zarf dev lint` integration)
- [Git](https://git-scm.com) (2.17.0 or later)
- [Kubectl](https://kubernetes.io/docs/reference/kubectl/overview/) (for deployment testing)
- Go 1.21+ (for building from source)

### Binary Distribution

Download the release distribution for your OS from the [Releases page](https://github.com/cpepper96/zarf-testing/releases).

Unpack the `zt` binary and add it to your PATH:

```bash
tar -xzf zt_linux_amd64.tar.gz
sudo mv zt /usr/local/bin/
zt --help
```

### From Source

```bash
git clone https://github.com/cpepper96/zarf-testing.git
cd zarf-testing
go build -o zt ./zt
./zt --help
```

## 🎯 Quick Start

### Basic Linting

```bash
# Lint all packages in the packages/ directory
zt lint --all

# Lint specific packages
zt lint --packages packages/my-package,packages/another-package

# Lint only changed packages (default)
zt lint
```

### Deployment Testing

```bash
# Test deployment of changed packages
zt install

# Test specific packages
zt install --packages packages/my-package

# Combined lint and install
zt lint-and-install --all
```

### Output Formats

```bash
# Colored output (default)
zt lint --packages packages/my-package

# JSON output for automation
zt lint --packages packages/my-package --output json

# GitHub Actions format
zt lint --packages packages/my-package --output github --github-groups
```

## ⚙️ Configuration

Create a `zt.yaml` file in your project root:

```yaml
# Zarf Testing Configuration
zarf-dirs:
  - packages
  - examples
remote: origin
target-branch: main

# Validation options
check-version-increment: true
validate-image-pinning: true
validate-package-schema: true
validate-components: true

# Deployment testing
deployment-timeout: 15m
test-timeout: 10m
skip-clean-up: false

# Output options
github-groups: false
```

### Environment Variables

All configuration options can be set via environment variables with the `ZT_` prefix:

```bash
export ZT_ZARF_DIRS="packages,examples"
export ZT_TARGET_BRANCH="main"
export ZT_CHECK_VERSION_INCREMENT="true"
```

## 📋 Commands

### `zt lint`

Validates Zarf packages using both the Zarf CLI and advanced custom rules.

**Validation Rules:**
- ✅ Basic Zarf package structure (`zarf dev lint`)
- ✅ Version increment when components change
- ✅ Image digest pinning enforcement
- ✅ Component naming conventions
- ✅ Component dependency validation
- ✅ Security best practices
- ✅ Resource constraint analysis

```bash
# Lint changed packages
zt lint

# Lint all packages
zt lint --all

# Lint specific packages
zt lint --packages packages/app,packages/db

# Custom validation options
zt lint --check-version-increment=false --validate-image-pinning=true
```

### `zt install`

Deploys and tests Zarf packages in a Kubernetes cluster.

**Testing Phases:**
1. 🔧 Package building with `zarf package create`
2. 🚀 Package deployment with `zarf package deploy`
3. ✅ Component validation and health checks
4. 🧹 Cleanup (optional with `--skip-clean-up`)

```bash
# Test changed packages
zt install

# Test specific packages
zt install --packages packages/my-app

# Skip cleanup for debugging
zt install --skip-clean-up

# Use custom namespace
zt install --namespace my-test-namespace
```

### `zt list-changed`

Lists packages that have changed compared to the target branch.

```bash
# List changed packages
zt list-changed

# Compare against specific branch
zt list-changed --target-branch develop

# Compare against specific remote
zt list-changed --remote upstream
```

## 🔍 Advanced Validation Rules

### Component Validation
- **Naming Conventions**: Lowercase, hyphen-separated names
- **Duplicate Detection**: Prevents duplicate component names
- **Empty Components**: Warns about components with no content
- **Required vs Default**: Flags redundant configuration

### Dependency Validation
- **Existence Checks**: Ensures all dependencies exist
- **Circular Dependencies**: Detects and prevents circular references
- **Self-Dependencies**: Prevents components from depending on themselves

### Security Validation
- **Privileged Containers**: Detects `privileged: true` in manifests
- **Host Networking**: Flags `hostNetwork: true` usage
- **Secret Detection**: Identifies potential hardcoded secrets in scripts
- **Registry Trust**: Warns about images from untrusted registries

### Resource Validation
- **Large Files**: Warns about files >100MB
- **Image Count**: Flags components with excessive images
- **Resource Limits**: Checks for missing CPU/memory limits

## 🎨 Output Formats

### Text Output (Default)
```
📋 Zarf Package Linting
ℹ️ Testing specified packages: [packages/my-app]
🔧 [██████████████████████████] 100% (1/1) Testing complete
✅ All packages passed validation
```

### JSON Output
```json
{
  "timestamp": "2025-07-27T23:44:34Z",
  "events": [
    {
      "type": "info",
      "message": "Testing specified packages: [packages/my-app]",
      "timestamp": "2025-07-27T23:44:34Z"
    },
    {
      "type": "success",
      "message": "All packages passed validation",
      "timestamp": "2025-07-27T23:44:34Z"
    }
  ]
}
```


## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📖 Examples

### Package Structure
```
my-zarf-package/
├── zarf.yaml
├── manifests/
│   └── deployment.yaml
├── files/
│   └── config.json
└── charts/
    └── my-chart/
```

### Validation Results
```bash
$ zt lint --packages packages/my-app

📋 Zarf Package Linting
ℹ️ Testing specified packages: [packages/my-app]

==> Linting packages/my-app
[WARNING] Issues found:
  - Component 'My App' doesn't follow naming conventions (use lowercase, hyphens)
  - Image not pinned with digest - nginx:latest
  - Component 'web' uses image from potentially untrusted registry: docker.io/nginx:latest
[INFO] Package validation successful (with warnings)

✅ All packages linted successfully
```

## 🆘 Troubleshooting

### Common Issues

**"zarf CLI not found"**
```bash
# Install Zarf CLI
curl -sL https://install.zarf.dev | bash
```

**"kubectl not available"**
```bash
# For deployment testing, ensure kubectl is configured
kubectl cluster-info
```

**"package not found"**
```bash
# Ensure package path contains zarf.yaml
ls packages/my-package/zarf.yaml
```

### Debug Mode

```bash
# Enable debug output
zt lint --debug

# Print configuration
zt lint --print-config
```

## 📜 License

This project is licensed under the Apache 2.0 License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Helm Chart Testing](https://github.com/helm/chart-testing) - Original inspiration and architecture
- [Zarf](https://zarf.dev)

## 🔗 Links

- [Zarf Documentation](https://docs.zarf.dev)
- [Zarf GitHub](https://github.com/zarf-dev/zarf)
- [Issues](https://github.com/cpepper96/zarf-testing/issues)
- [Discussions](https://github.com/cpepper96/zarf-testing/discussions)
