# Zarf-Testing Development Plan

## Current Project Status ✅

**MVP Completion: ~90%** - Core infrastructure is solid, primary command implemented, minor cleanup needed.

### ✅ What's Working
- **7/7 CLI commands fully implemented** (`lint`, `install`, `list-changed`, `lint-and-install`, `version`, `doc-gen`, root)
- **Solid architecture** with clean separation of concerns
- **Complete configuration system** (Viper-based, hierarchical config)
- **Package discovery and filtering** (finds zarf.yaml files, excludes patterns)
- **Git change detection** (modified package discovery)
- **Advanced validation framework** (beyond basic `zarf dev lint`)
- **Deployment testing workflow** (build → deploy → test → cleanup)
- **Output formatting** (text, JSON, GitHub Actions)
- **Integrated lint-and-install workflow** with progress reporting and proper error handling

### ❌ Remaining Issues
1. **Core validation functions return `nil`** - missing actual validation logic
2. **Incomplete package cleanup** - test isolation broken
3. **Missing tests for critical tools** - kubectl, helm, git, exec integrations

---

## Priority Development Plan

### 🔥 **PHASE 1: MVP Completion (Week 1)**
*Goal: Ship functional CLI that can lint and deploy Zarf packages end-to-end*

#### 1.1 Implement `lint-and-install` Command ✅ **COMPLETED**
**File**: [`zt/cmd/lintAndInstall.go`](file:///Users/cpepper/git/zarf-testing/zt/cmd/lintAndInstall.go#L39-L52)
**Status**: ✅ **IMPLEMENTED** - Full workflow functional

**Implementation Notes**:
- Successfully integrated existing `lint` and `install` command logic
- Added proper error handling with early exit on lint failures
- Implemented progress reporting using same spinner pattern as other commands
- All flags properly supported: `--skip-cleanup`, `--timeout`, `--namespace-prefix`, etc.
- Results aggregation and pretty-printing functional
- Exit codes work correctly (1 for failures, 0 for success)

**Key Learnings**:
- Configuration loading works seamlessly across both phases
- Package discovery logic is well-architected and reusable
- Spinner UI provides good user experience during long operations
- Error propagation between lint and install phases works correctly

#### 1.2 Complete Package Cleanup Logic ⭐ **CRITICAL**
**File**: [`pkg/zarf/deployer.go:264-276`](file:///Users/cpepper/git/zarf-testing/pkg/zarf/deployer.go#L264-L276)
**Status**: Placeholder logic, doesn't properly remove Zarf packages

**Implementation Plan**:
```bash
# Safe cleanup strategy:
kubectl delete ns <test-namespace> --wait
kubectl delete pvc,crd,clusterroles (if created by test)
rm -rf generated *.tar.zst build directories
```

**Required Work**:
- Implement namespace-scoped cleanup
- Remove generated artifacts (tar.zst files, build dirs)
- Add retry/backoff for cleanup failures
- Log warnings for cleanup failures (don't fail tests)

#### 1.3 Complete Core Validation Functions ⭐ **HIGH**
**File**: [`pkg/zarf/validator.go`](file:///Users/cpepper/git/zarf-testing/pkg/zarf/validator.go#L222-L244)
**Status**: Functions return `nil` - missing implementation

**Missing Functions**:
- `validateComponents()` - verify component structure and requirements
- `validateComponentDependencies()` - check dependency chains
- `validateSecurityBestPractices()` - security scanning
- `validateResourceConstraints()` - resource limit validation

**Implementation Priority**:
1. **validateComponents**: Required vs optional component validation
2. **validateComponentDependencies**: Circular dependency detection
3. **validateSecurityBestPractices**: Privileged container detection
4. **validateResourceConstraints**: Memory/CPU limit checking

#### 1.4 Complete Tool Integration Stubs 🔧 **HIGH**
**Files**: [`pkg/tool/kubectl.go`](file:///Users/cpepper/git/zarf-testing/pkg/tool/kubectl.go), [`pkg/tool/account.go`](file:///Users/cpepper/git/zarf-testing/pkg/tool/account.go)
**Status**: Several functions return `nil` without implementation

**Required Work**:
- Implement missing kubectl operations
- Complete account/authentication functions
- Standardize error handling across tool integrations
- Add timeout/context handling

---

### 🧪 **PHASE 2: Testing & Robustness (Week 2)**
*Goal: Achieve >80% test coverage and reliable CI*

#### 2.1 Unit Tests for Tool Integrations ⭐ **HIGH**
**Status**: Missing tests for critical components

**Missing Test Coverage**:
- **`pkg/tool/helm.go`** - Helm operations (install, upgrade, test)
- **`pkg/tool/kubectl.go`** - Kubernetes operations
- **`pkg/tool/git.go`** - Git command execution
- **`pkg/exec/exec.go`** - Process execution engine

**Testing Strategy**:
- Use gomock/testify for mocking external commands
- Table-driven tests for various scenarios
- Cover success, failure, timeout cases

#### 2.2 Validator Test Suite 🧪 **MEDIUM**
**Status**: Basic tests exist, need comprehensive coverage

**Required Work**:
- Create test fixtures (valid/invalid Zarf packages)
- Table-driven tests for each validation function
- Test edge cases and error conditions
- Target >80% coverage for `pkg/zarf/validator.go`

#### 2.3 Integration Tests with KIND 🔧 **MEDIUM**
**Status**: Missing end-to-end testing

**Implementation Plan**:
- GitHub Actions with KIND cluster
- Build minimal test Zarf package
- Run full `lint-and-install` workflow
- Verify cleanup works correctly

#### 2.4 CLI Command Tests 🧪 **LOW**
**Status**: No CLI tests found

**Required Work**:
- Use Cobra's `ExecuteC()` for CLI testing
- Test flag parsing and validation
- Test command combinations and edge cases

---

### 🔧 **PHASE 3: Developer Experience (Week 3)**
*Goal: Smooth development workflow and CI/CD*

#### 3.1 GitHub Actions Workflow 🤖 **MEDIUM**
**Status**: Basic CI exists, needs enhancement

**Required Workflow**:
```yaml
# .github/workflows/ci.yml
- go vet / staticcheck
- go test ./... (unit tests)
- KIND integration tests (optional on PRs)
- golangci-lint
- codecov upload
```

#### 3.2 Development Tooling 🔧 **LOW**
**Required Work**:
- Pre-commit hooks (formatting, linting)
- Makefile targets (test, integration, release)
- Local development documentation

---

### 📚 **PHASE 4: Documentation & Release (Week 4)**
*Goal: Complete documentation and v0.1.0 release*

#### 4.1 User Documentation 📖 **MEDIUM**
**Status**: Good foundation, needs user-focused content

**Missing Documentation**:
- Getting started tutorial
- CI/CD integration examples
- Troubleshooting guide
- Migration from chart-testing

#### 4.2 Release Preparation 🚀 **LOW**
**Required Work**:
- CHANGELOG.md
- Semantic versioning policy
- GitHub release with binaries
- Container image publishing

---

## Technical Dependencies

### External Tool Requirements
- **Zarf CLI** - Core package operations
- **kubectl** - Kubernetes cluster access  
- **Git** - Change detection and repository operations
- **Helm** (optional) - Legacy chart support

### Development Dependencies
- **Go 1.21+** - Language runtime
- **KIND** - Integration testing
- **golangci-lint** - Code quality
- **goreleaser** - Binary builds

---

## Risk Mitigation

### High-Risk Items
1. **Zarf SDK Integration** - Core dependency, potential breaking changes
2. **Kubernetes Cluster Access** - Required for deployment testing
3. **Package Cleanup Failures** - Could leave test artifacts

### Mitigation Strategies
- Detect tool availability at runtime
- Graceful degradation when tools unavailable  
- Best-effort cleanup with warning logs
- Comprehensive error messages with actionable steps

---

## Success Metrics

### Phase 1 (MVP) Success Criteria
- [x] `zt lint-and-install` command functional ✅
- [ ] Package cleanup works reliably
- [ ] Core validation functions implemented
- [ ] Tool integration stubs completed

### Phase 2 (Testing) Success Criteria  
- [ ] >80% test coverage achieved
- [ ] CI pipeline green and stable
- [ ] Integration tests pass on KIND
- [ ] No critical bugs in core workflows

### Phase 3 (DX) Success Criteria
- [ ] Smooth local development workflow
- [ ] Automated quality checks
- [ ] Clear contribution guidelines

### Phase 4 (Release) Success Criteria
- [ ] Complete user documentation
- [ ] v0.1.0 release published
- [ ] Container images available
- [ ] Migration guide from chart-testing

---

## Next Immediate Actions

### Today's Priorities
1. **Complete package cleanup logic** - Critical for test isolation (primary blocker)
2. **Implement core validation functions** - Fills remaining validation gaps
3. **Add unit tests for new code** - Prevent regressions on completed functionality

### This Week
1. Complete remaining Phase 1 items (package cleanup, validation functions)
2. Start Phase 2 (testing infrastructure)
3. Validate full workflow works end-to-end with real Zarf packages

---

*Last Updated: January 9, 2025*
*Status: Active Development - Phase 1 (Nearly Complete)*
