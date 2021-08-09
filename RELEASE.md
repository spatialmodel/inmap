# Release checklist

This file contains a checklist for steps to take to release a new version of InMAP.

1. Update the version number in `framework.go`.

1. Change the version number in `website/static/2019-04-20-sr/sr_util.py`.

1. Update the version number in `README.md`. Make sure the README and other documentation is up to date.

1. If the input data format has changed since the last release, change the `DataVersion` and/or `VarGridDataVersion` variables in `framework.go` and regenerate the input data with the new version number.

1. Commit the results.

1. Tag the release version using `git tag v${version}` and push it using `git push origin v${version}`.

1. Create a release on github and add any new input or evaluation data as downloads.

1. Github actions will automatically add precompiled binaries to the release.