# Test Charts Directory

**Note**: This directory contains legacy Helm charts from the chart-testing project that are used for internal testing of the ignore functionality and other chart-testing compatibility features.

These charts are not used for Zarf package testing functionality and can be safely ignored by users of zarf-testing.

For actual Zarf package examples, see the `packages/` directory in the repository root.

## Contents

- Various Helm charts used for internal testing
- Used primarily by `pkg/ignore` tests
- Maintains compatibility with chart-testing architecture

## Future

These may be converted to Zarf packages or removed in a future version as we fully migrate away from chart-testing dependencies.
