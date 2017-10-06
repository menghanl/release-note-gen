# release-note-gen

`release-note-gen` generates release notes for a release from closed github PRs
within the corresponding release milestone. For example, PRs that should be
included in release note for 1.7.0 are all set to milestone "1.7 Release".

For each closed PR, it generates one line release note in for form of:
```
description (#<pr>)
```
For example:
```
Add grpc.Version string and use it in the UA (#1144)
```
 
The PR's "Type" labels are used as the category of the change.
For example, a PR with label `["Type: API change"]` will be classified as `"API change"`.

- If a PR has more than one labels, the labels will be sorted in the order of
 `"Dependencies", "API Change", "Feature", "Behavior Change", "Performance", "Bug", "Internal Cleanup", "Documentation", "Testing"`,
 and the first one will be picked as the final category.
- If a PR has no "Type" label (which shouldn't happen), it will be classified as
  `"Bug"`.
