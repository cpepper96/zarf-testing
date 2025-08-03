## zt lint-and-install

Lint, install, and test a Zarf package

### Synopsis

Combines 'lint' and 'install' commands for Zarf packages.

```
zt lint-and-install [flags]
```

### Options

```
      --additional-commands strings   Additional commands to run per package (default: [])
                                      Commands will be executed in the same order as provided in the list and will
                                      be rendered with go template before being executed.
                                      Example: "zarf package inspect {{ .Path }}"
      --all                           Process all packages except those explicitly excluded.
                                      Disables changed package detection and version increment checking
      --build-id string               An optional, arbitrary identifier that is added to the name of the namespace a
                                      package is installed into. In a CI environment, this could be the build number or
                                      the ID of a pull request. If not specified, the name of the package is used
      --check-version-increment       Activates a check for package version increments (default true)
      --config string                 Config file
      --debug                         Print CLI calls of external tools to stdout
      --exclude-deprecated            Skip packages that are marked as deprecated
      --excluded-packages strings     Packages that should be skipped. May be specified multiple times
                                      or separate values with commas
      --github-groups                 Change the delimiters for github to create collapsible groups
                                      for command output
  -h, --help                          help for lint-and-install
      --lint-conf string              The config file for YAML linting. If not specified, 'lintconf.yaml'
                                      is searched in the current directory, '$HOME/.zt', and '/etc/zt', in
                                      that order
      --namespace string              Namespace to install the release(s) into. If not specified, each release will be
                                      installed in its own randomly generated namespace
      --no-color                      Disable colored output
      --output string                 Output format: text, json, github (default "text")
      --packages strings              Specific packages to test. Disables changed package detection and
                                      version increment checking. May be specified multiple times
                                      or separate values with commas
      --print-config                  Prints the configuration to stderr
      --release-name string           Name for the release. If not specified, is set to the package name and a random 
                                      identifier.
      --remote string                 The name of the Git remote used to identify changed packages (default "origin")
      --since string                  The Git reference used to identify changed packages (default "HEAD")
      --skip-clean-up                 Skip resources clean-up after testing
      --skip-missing-values           When --upgrade has been passed, this flag will skip testing CI values files from the
                                      previous package revision if they have been deleted or renamed at the current package
                                      revision
      --target-branch string          The name of the target branch used to identify changed packages (default "main")
      --upgrade                       Whether to test an in-place upgrade of each package from its previous revision if the
                                      current version should not introduce a breaking change according to the SemVer spec
      --validate-yaml                 Enable linting of 'zarf.yaml' and configuration files (default true)
      --zarf-dirs strings             Directories containing Zarf packages. May be specified multiple times
                                      or separate values with commas (default [packages])
```

### SEE ALSO

* [zt](zt.md)	 - The Zarf package testing tool

