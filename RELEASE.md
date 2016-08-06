# Release checklist

This file contains a checklist for steps to take to release a new version of InMAP.

1. Update vendored packages using `govendor fetch`.

1. Update the version number in `framework.go` and in `inmap/build.sh`, and the `year` variable in `inmap/cmd/root.go`.

1. If the input data format has changed since the last release, change the `DataVersion` and/or `VarGridDataVersion` variables in `framework.go` and regenerate the input data with the new version number.

1. Commit the results.

1. Run `inmap/build.sh` to create executables for different platforms.

1. Tag the release version using `git tag v${version}` and push it using `git push origin v${version}`.

1. Create a release on github and add the binary executables and any new input or evaluation data as downloads.
