## zt list-changed

List changed packages

### Synopsis

"List changed Zarf packages based on configured package directories,
"remote, and target branch

```
zt list-changed [flags]
```

### Options

```
      --config string               Config file
      --exclude-deprecated          Skip packages that are marked as deprecated
      --excluded-packages strings   Packages that should be skipped. May be specified multiple times
                                    or separate values with commas
      --github-groups               Change the delimiters for github to create collapsible groups
                                    for command output
  -h, --help                        help for list-changed
      --no-color                    Disable colored output
      --output string               Output format: text, json, github (default "text")
      --print-config                Prints the configuration to stderr
      --remote string               The name of the Git remote used to identify changed packages (default "origin")
      --since string                The Git reference used to identify changed packages (default "HEAD")
      --target-branch string        The name of the target branch used to identify changed packages (default "main")
      --zarf-dirs strings           Directories containing Zarf packages. May be specified multiple times
                                    or separate values with commas (default [packages])
```

### SEE ALSO

* [zt](zt.md)	 - The Zarf package testing tool

