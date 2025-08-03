## zt lint

Lint and validate a Zarf package

### Synopsis

Run Zarf package validation, version checking, YAML schema validation
on 'zarf.yaml', YAML linting on package configuration files,
and component validation on

* changed packages (default)
* specific packages (--packages)
* all packages (--all)

in given package directories.

Packages may have multiple custom configuration files in the package
directory. The package is linted and validated according to Zarf
package specifications and component requirements.

```
zt lint [flags]
```

### Options

```
      --additional-commands strings   Additional commands to run per package (default: [])
                                      Commands will be executed in the same order as provided in the list and will
                                      be rendered with go template before being executed.
                                      Example: "zarf package inspect {{ .Path }}"
      --all                           Process all packages except those explicitly excluded.
                                      Disables changed package detection and version increment checking
      --check-version-increment       Activates a check for package version increments (default true)
      --config string                 Config file
      --debug                         Print CLI calls of external tools to stdout
      --exclude-deprecated            Skip packages that are marked as deprecated
      --excluded-packages strings     Packages that should be skipped. May be specified multiple times
                                      or separate values with commas
      --github-groups                 Change the delimiters for github to create collapsible groups
                                      for command output
  -h, --help                          help for lint
      --lint-conf string              The config file for YAML linting. If not specified, 'lintconf.yaml'
                                      is searched in the current directory, '$HOME/.zt', and '/etc/zt', in
                                      that order
      --no-color                      Disable colored output
      --output string                 Output format: text, json, github (default "text")
      --packages strings              Specific packages to test. Disables changed package detection and
                                      version increment checking. May be specified multiple times
                                      or separate values with commas
      --print-config                  Prints the configuration to stderr
      --remote string                 The name of the Git remote used to identify changed packages (default "origin")
      --since string                  The Git reference used to identify changed packages (default "HEAD")
      --target-branch string          The name of the target branch used to identify changed packages (default "main")
      --validate-yaml                 Enable linting of 'zarf.yaml' and configuration files (default true)
      --zarf-dirs strings             Directories containing Zarf packages. May be specified multiple times
                                      or separate values with commas (default [packages])
```

### SEE ALSO

* [zt](zt.md)	 - The Zarf package testing tool

