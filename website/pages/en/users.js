/**
 * Copyright (c) 2017-present, Facebook, Inc.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

const React = require('react');

const CompLibrary = require('../../core/CompLibrary.js');

const Container = CompLibrary.Container;

const siteConfig = require(`${process.cwd()}/siteConfig.js`);

class Users extends React.Component {
  render() {
    if ((siteConfig.users || []).length === 0) {
      return null;
    }

    const editUrl = `${siteConfig.repoUrl}/edit/master/website/siteConfig.js`;
    const showcase = siteConfig.users.map(user => (
      <a href={user.infoLink} key={user.infoLink}>
        <figure>
          <img src={user.image} alt={user.caption} title={user.caption} />
          <figcaption>{user.caption}</figcaption>
          </figure>
      </a>
    ));

    return (
      <div className="mainContainer">
        <Container padding={['bottom', 'top']}>
          <div className="showcaseSection">
            <div className="prose">
              <h1>InMAP has been used to:</h1>
            </div>
            <div className="logos">{showcase}</div>
            <h3>Are you using InMAP?</h3>
            <a href={editUrl} className="button">
              Add your project
            </a>
          </div>
        </Container>
      </div>
    );
  }
}

module.exports = Users;
