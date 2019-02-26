/**
 * Copyright (c) 2017-present, Facebook, Inc.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

// See https://docusaurus.io/docs/site-config for all the possible
// site configuration options.

// List of projects/orgs using your project for the users page.
const users = [
  {
    caption: 'Study environmental justice in California',
    // You will need to prepend the image path with your baseUrl
    // if it is not '/', like: '/test-site/img/docusaurus.svg'.
    image: 'https://www.ucsusa.org/sites/default/files/styles/reports_thumbnail/public/images/2019/01/air-quality-eng-cover.JPG?itok=S5hegygC',
    infoLink: 'https://www.ucsusa.org/clean-vehicles/electric-vehicles/CA-air-quality-equity',
    pinned: true,
  },
  {
    caption: 'Estimate health impacts of shipping',
    // You will need to prepend the image path with your baseUrl
    // if it is not '/', like: '/test-site/img/docusaurus.svg'.
    image: 'https://media.springernature.com/w200/springer-static/cover-hires/journal/41893/2/2',
    infoLink: 'https://www.nature.com/natsustain/volumes/2/issues/2',
    pinned: true,
  },
];

const siteConfig = {
  title: 'InMAP', // Title for your website.
  tagline: 'INTERVENTION MODEL FOR AIR POLLUTION',
  url: 'https://inmap.run', // Your website URL
  baseUrl: '/', // Base URL for your project */
  // For github.io type URLs, you would set the url and baseUrl like:
  //   url: 'https://facebook.github.io',
  //   baseUrl: '/test-site/',

  // Used for publishing and more
  projectName: 'InMAP',
  organizationName: 'The InMAP authors',
  // For top-level user or org sites, the organization is still the same.
  // e.g., for the https://JoelMarcey.github.io site, it would be set like...
  //   organizationName: 'JoelMarcey'

  // For no header links in the top nav bar -> headerLinks: [],
  headerLinks: [
    {doc: 'quickstart', label: 'Docs'},
    {href: 'https://inmap.run/eieio', label: 'EIEIO'},
    {href: 'https://godoc.org/github.com/spatialmodel/inmap', label: 'API'},
    {page: 'help', label: 'Help'},
    {blog: true, label: 'Blog'},
  ],

  // If you have users set above, you add it here:
  users,

  disableHeaderTitle: true,

  /* path to images for header/footer */
  headerIcon: 'img/textLogo.svg',
  footerIcon: 'img/textLogo.svg',
  favicon: 'img/favicon/favicon.ico',

  /* Colors for website */
  colors: {
    primaryColor: '#353535',
    secondaryColor: '#777777',
  },

  /* Custom fonts for website */
  /*
  fonts: {
    myFont: [
      "Times New Roman",
      "Serif"
    ],
    myOtherFont: [
      "-apple-system",
      "system-ui"
    ]
  },
  */

  // This copyright info is used in /core/Footer.js and blog RSS/Atom feeds.
  copyright: `Copyright © ${new Date().getFullYear()} the InMAP authors.`,

  highlight: {
    // Highlight.js theme to use for syntax highlighting in code blocks.
    theme: 'default',
  },

  // Add custom scripts here that would be placed in <script> tags.
  scripts: ['https://buttons.github.io/buttons.js'],

  // On page navigation for the current documentation page.
  onPageNav: 'separate',
  // No .html extensions for paths.
  cleanUrl: true,

  // Open Graph and Twitter card images.
  ogImage: 'img/logo.svg',
  twitterImage: 'img/logo.svg',

  // You may provide arbitrary config keys to be used as needed by your
  // template. For example, if you need your repo's URL...
  repoUrl: 'https://github.com/spatialmodel/inmap',
};

module.exports = siteConfig;
