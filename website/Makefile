# Development targets for the InMAP website. The published website is built
# and deployed by the 'website' GitHub Actions workflow; these targets are for
# previewing changes locally. See README.md in this directory.

# The command documentation in ../docs is automatically generated and is not
# checked in, so it has to be created before the website can be built.
docs :
	cd .. && go generate .

node_modules : package-lock.json
	npm ci

# Build the website into build/InMAP.
build : docs node_modules
	npx docusaurus-build

# Serve the website at localhost:3000.
host : docs node_modules
	npx docusaurus-start

clean :
	rm -rf node_modules build

.PHONY : docs build host clean
