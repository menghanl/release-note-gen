# release-note-gen

`release-note-gen` generates release notes for a release from closed github PRs.
It uses all closed PRs labelled as part of the release to generate the note.

For each closed PR, it generates one line release note in for form of:
```
description (#<pr>)
```
For example:
```
Add grpc.Version string and use it in the UA (#1144)
```
 
It uses the PR labels other than release number as the category of the change.

- If a PR has label `["1.3", "API change"]`, it will be classified as `"API change"`.
- If a PR has more than one labels, the labels will be sorted in the order of
 `"API change", "New Feature", "Behavior change", "Bug fix", "Performance", "Documentation"`
- If a PR has only the release number label, it will be classified as `"Bug fix"`.
