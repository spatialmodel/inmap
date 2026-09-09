# The InMAP website

This directory holds the source for the InMAP website at https://inmap.run,
which is built with [Docusaurus](https://docusaurus.io/) version 1.

The website content lives in `../docs` (documentation pages), `blog` (blog
posts), `pages` (standalone pages), and `static` (images, videos, and other
assets). `siteConfig.js` and `sidebars.json` configure the site.

## Deployment

The website is built and deployed to GitHub Pages by the `website` GitHub
Actions workflow (`../.github/workflows/website.yml`) every time changes land
on the `master` branch. Pull requests build the website as well, but do not
deploy it.

Neither the generated website in `build` nor the installed dependencies in
`node_modules` are checked in; both are created by the workflow. The same is
true of the command documentation in `../docs/cmd` and
`../docs/output_options.md`, which are generated from the InMAP source code by
`go generate .` in the repository root.

## Building locally

To serve the website at http://localhost:3000, with the command documentation
generated first:

```sh
make host
```

To just build the website into `build/InMAP`:

```sh
make build
```

`make clean` removes `node_modules` and `build`.
