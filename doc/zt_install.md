## zt install

Install and test a Zarf package

### Synopsis

Deploy and test Zarf packages on

* changed packages (default)
* specific packages (--packages)
* all packages (--all)

in given package directories. This command will deploy packages
and validate that all components are working correctly.

Packages will be deployed to test namespaces and validated
for proper functionality. If no test configuration is present,
the package is deployed and tested with defaults.

```
zt install [flags]
```

### Options

```
      --all                         Process all packages except those explicitly excluded.
                                    Disables changed package detection and version increment checking
      --build-id string             An optional, arbitrary identifier that is added to the name of the namespace a
                                    package is installed into. In a CI environment, this could be the build number or
                                    the ID of a pull request. If not specified, the name of the package is used
      --config string               Config file
      --debug                       Print CLI calls of external tools to stdout
      --exclude-deprecated          Skip packages that are marked as deprecated
      --excluded-packages strings   Packages that should be skipped. May be specified multiple times
                                    or separate values with commas
      --github-groups               Change the delimiters for github to create collapsible groups
                                    for command output
  -h, --help                        help for install
      --namespace string            Namespace to install the release(s) into. If not specified, each release will be
                                    installed in its own randomly generated namespace
      --no-color                    Disable colored output
      --output string               Output format: text, json, github (default "text")
      --packages strings            Specific packages to test. Disables changed package detection and
                                    version increment checking. May be specified multiple times
                                    or separate values with commas
      --print-config                Prints the configuration to stderr
      --release-name string         Name for the release. If not specified, is set to the package name and a random 
                                    identifier.
      --remote string               The name of the Git remote used to identify changed packages (default "origin")
      --since string                The Git reference used to identify changed packages (default "HEAD")
      --skip-clean-up               Skip resources clean-up after testing
      --skip-missing-values         When --upgrade has been passed, this flag will skip testing CI values files from the
                                    previous package revision if they have been deleted or renamed at the current package
                                    revision
      --target-branch string        The name of the target branch used to identify changed packages (default "main")
      --upgrade                     Whether to test an in-place upgrade of each package from its previous revision if the
                                    current version should not introduce a breaking change according to the SemVer spec
      --zarf-dirs strings           Directories containing Zarf packages. May be specified multiple times
                                    or separate values with commas (default [packages])
```

### SEE ALSO

* [zt](zt.md)	 - The Zarf package testing tool

