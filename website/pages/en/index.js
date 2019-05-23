/**
 * Some of this file is Copyright (c) 2017-present, Facebook, Inc.
 * Other parts are Copyright the InMAP authors.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

const React = require('react');

const CompLibrary = require('../../core/CompLibrary.js');

const MarkdownBlock = CompLibrary.MarkdownBlock; /* Used to read markdown */
const Container = CompLibrary.Container;
const GridBlock = CompLibrary.GridBlock;

const siteConfig = require(`${process.cwd()}/siteConfig.js`);

function imgUrl(img) {
  return `${siteConfig.baseUrl}img/${img}`;
}

function docUrl(doc, language) {
  return `${siteConfig.baseUrl}docs/${language ? `${language}/` : ''}${doc}`;
}

function pageUrl(page, language) {
  return siteConfig.baseUrl + (language ? `${language}/` : '') + page;
}

class Button extends React.Component {
  render() {
    return (
      <div className="pluginWrapper buttonWrapper">
        <a className="button" href={this.props.href} target={this.props.target}>
          {this.props.children}
        </a>
      </div>
    );
  }
}

Button.defaultProps = {
  target: '_self',
};

const SplashContainer = props => (
  <div className="homeContainer" id="home">
    <div className="homeSplashFade">
      <div className="wrapper homeWrapper">{props.children}</div>
    </div>
  </div>
);

const Logo = props => (
  <div className="projectLogo">
    <img src={props.img_src} alt="Project Logo" />
  </div>
);

const BootstrapContainer = props => (
  <div className="container">
    {props.children}
  </div>
);

const ProjectTitle = () => (
  <div>
    <h1>INTERVENTION MODEL FOR AIR POLLUTION</h1>
    <h2>Health Impacts of Air Pollution: A Tool to<br />Understand the Consequences</h2>
    <h3>Christopher Tessum | Jason Hill | Julian Marshall</h3>
  </div>
);

const Publication = () => (
  <div className="row">
      <div className="col-md-12">
          <p><strong>Publication:</strong>
          Tessum, C. W.; Hill, J. D.; Marshall, J. D. InMAP: A model for air pollution interventions.
          <em>PLoS ONE</em> <strong>2017</strong>, <em>12</em> (4), e0176131 DOI:
          <a href="https://doi.org/10.1371/journal.pone.0176131"> 10.1371/journal.pone.0176131</a>.</p>
      </div>
  </div>
);

const PromoSection = props => (
  <div className="section promoSection">
    <div className="promoRow">
      <div className="pluginRowBlock">{props.children}</div>
    </div>
  </div>
);

class HomeSplash extends React.Component {
  render() {
    const language = this.props.language || '';
    return (
      <SplashContainer>
      <img src="img/logo.svg" alt="InMAP logo" />
        <div className="inner">
          <ProjectTitle />
          <Publication />
          <PromoSection>
              <Button href="#home">Home</Button>
              <Button href="#about">About</Button>
              <Button href="#methodology">Methodology</Button>
              <Button href="#video">Video</Button>
              <Button href="#download">Download</Button>
          </PromoSection>
        </div>
      </SplashContainer>
    );
  }
}

const HR = ({ id }) => (
    <hr id={id}/>
);

const Block = props => (
  <Container
    padding={['bottom', 'top']}
    id={props.id}
    background={props.background}>
    <GridBlock align="center" contents={props.children} layout={props.layout} />
  </Container>
);

const WhatIs = () => (
  <div className="row">
      <div className="col-md-4">
          <h2> What is <object data="img/textLogo.svg" type="image/svg+xml">InMAP</object>?</h2>
          <p>InMAP is a recently developed model which offers a new approach to estimating the human health impacts caused by air pollutant emissions and how those impacts are distributed among different groups of people. InMAP is:</p>
      </div>
  </div>
);

const Description = () => (
  <div className="row">
      <div className="col-md-4">
          <h4>Accessible</h4>
          <p>Designed to be used by a wide range of professionals concerned with how air pollution affects health.</p>
          <h4>Comprehensive</h4>
          <p>One tool to perform an entire health impact analysis.
          </p>
          <h4>Fast</h4>
          <p>Runs on a desktop computer instead of a supercomputer as usually required by comprehensive air quality models.</p>
          <h4>Big + small</h4>
          <p>Able to simultaneously track within-city differences in impacts as well as impacts thousands of miles from the emissions source, unlike other simplified models.</p>
          <h4>Accurate</h4>
          <p>Meets published criteria for air quality model predictive performance.</p>
      </div>
      <div className="col-md-8">
          <img src="img/diagram1.svg" alt="InMAP diagram 1" className="img-responsive" />
          <p>InMAP allows users to explore the consequences of emissions changes at a high resolution in a simple and computationally inexpensive way.</p>
      </div>
  </div>
);

const Methodology = () => (
  <div className="row">
      <div className="col-md-4">
          <h2> Why use <object data="img/textLogo.svg" type="image/svg+xml">InMAP</object>?</h2>
          <p>Because InMAP is a reduced complexity air quality model, it may not be the perfect tool for every job. However, InMAP is well suited for many situations, such as:</p>
              <ul>
                  <li> Projects that require many model runs, such as those that include scenario or uncertainty assessment.</li>
                  <li> Projects that would benefit from the combination of a large spatial resolution an high spatial resolution compared to what is available in other models.</li>
                  <li> Projects interested in investigation environmental injustice or equity issues.</li>
                  <li> Projects that do not have access to the time, expertise, or resources required to run comprehensive chemical transport models.</li>
              </ul>
      </div>
      <div className="col-md-8">
          <img src="img/diagram2.svg" alt="InMAP diagram 2" className="img-responsive" />
      </div>
  </div>
);


const Video = () => (
  <div className="row">
    <div className="col-md-12">
      <figure className="figure">
        <figcaption className="figure-caption"><h2><object data="img/textLogo.svg" type="image/svg+xml">InMAP</object> in action</h2>
          This animation shows an InMAP model simulation. The model runs until it converges on a steady-state solution,
          adjusting the size of the grid cells as it runs.</figcaption>
          <div align="center" className="embed-responsive embed-responsive-16by9">
              <video controls autoPlay loop className="embed-responsive-item">
                <source src="vid/inmapNEI.mp4" type="video/mp4" />
                <source src="vid/inmapNEI.webm" type="video/webm" />
                <source src="vid/inmapNEI.ogv" type="video/ogg" />
                Your browser does not support the video tag.
              </video>
          </div>
      </figure>
    </div>
  </div>
);

const Download = () => (
  <div className="row">
      <div className="col-md-8">
          <h2> How is <object data="img/textLogo.svg" type="image/svg+xml">InMAP</object> new?</h2>
          <p>InMAP includes several features that together enable analyses to be done in InMAP that cannot be done or are much more difficult in other models. These features include:
              <ul>
                  <li> A variable-resolution computational grid, which allows the model to save time by focusing computational resources in areas where extra spatial detail is most useful.</li>
                  <li> Simplified chemistry and physics parameterizations that save computational time while still creating a mechanistic representation of the atmosphere.</li>
                  <li> Health impact calculations built into the model to avoid the need for a separate tool.</li>
                  <li> A software framework that allows the model to run on most types of computers.</li>
                  <li> Input and output files in the widely-used
                      <a href="https://en.wikipedia.org/wiki/Shapefile"> shapefile</a> format.
                  </li>
              </ul>
          </p>
      </div>
      <div className="col-md-4">
          <h2> How do I use <object data="img/textLogo.svg" type="image/svg+xml">InMAP</object>?</h2>
          <p>The InMAP model is completely open-source. It is available to download
              <a href="http://code.spatialmodel.com"> here</a> and instructions for its use are on the same page.</p>

          <p> Join the <a href="https://groups.google.com/forum/#!forum/inmap-users">InMAP users' Google Group</a> for news and discussions related to the model.</p>

          <p> Additional questions? Contact us at inmap@spatialmodel.com</p>
      </div>
  </div>
);

const OtherModels = () => (
  <div className="row">
      <div className="col-md-12">
          <h2>Other models</h2>
          <p>There are also a number of other reduced-complexity air quality models available:</p>
          <p><a href="https://public.tepper.cmu.edu/nmuller/APModel.aspx">AP3</a></p>
          <p><a href="http://barney.ce.cmu.edu/~jinhyok/easiur/">EASIUR</a></p>
      </div>
  </div>
);

class Index extends React.Component {
  render() {
    const language = this.props.language || '';

    return (
      <div>
        <HomeSplash language={language} />
        <div className="mainContainer">
          <BootstrapContainer>
          <HR id="about" />
          <WhatIs />
          <Description />
          <HR id="methodology" />
          <Methodology />
          <HR id="video" />
          <Video />
          <HR id="download" />
          <Download />
          <HR id="othermodels" />
          <OtherModels />
          </BootstrapContainer>
        </div>
      </div>
    );
  }
}

module.exports = Index;
