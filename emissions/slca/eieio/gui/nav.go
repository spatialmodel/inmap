/*
Copyright © 2017 the InMAP authors.
This file is part of InMAP.

InMAP is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

InMAP is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with InMAP.  If not, see <http://www.gnu.org/licenses/>.*/

package main

import (
	"github.com/gopherjs/vecty"
	"github.com/gopherjs/vecty/elem"
	"github.com/gopherjs/vecty/prop"
)

// Navigator represents a navigation bar.
type Navigator struct {
	vecty.Core
	Title string
}

// Render renders the navigation bar.
func (n *Navigator) Render() vecty.ComponentOrHTML {
	return elem.Navigation(vecty.Markup(vecty.Class("navbar", "navbar-inverse", "navbar-fixed-top")),
		elem.Div(vecty.Markup(vecty.Class("container-fluid")),
			elem.Div(vecty.Markup(vecty.Class("navbar-header")),
				elem.Button(
					vecty.Markup(
						vecty.Class("navbar-toggle", "collapsed"),
						vecty.Data("toggle", "collapse"),
						vecty.Data("target", "#navbar"),
					),
					elem.Span(vecty.Markup(vecty.Class("sr-only")),
						vecty.Text("Toggle navigation"),
					),
					elem.Span(vecty.Markup(vecty.Class("icon-bar"))),
					elem.Span(vecty.Markup(vecty.Class("icon-bar"))),
					elem.Span(vecty.Markup(vecty.Class("icon-bar"))),
				),
				elem.Anchor(vecty.Markup(vecty.Class("navbar-brand"), prop.Href("/")),
					elem.Image(vecty.Markup(prop.Src("/img/textLogo.svg"))),
				),
				elem.Anchor(vecty.Markup(vecty.Class("navbar-brand"), prop.Href("#")),
					vecty.Text(n.Title),
				),
			),
			elem.Div(vecty.Markup(prop.ID("navbar"), vecty.Class("collapse", "navbar-collapse")),
				elem.UnorderedList(
					vecty.Markup(vecty.Class("nav", "navbar-nav", "navbar-right")),
					elem.ListItem(
						elem.Anchor(vecty.Markup(prop.Href("/docs/quickstart")), vecty.Text("Docs")),
					),
					elem.ListItem(
						vecty.Markup(vecty.Class("active")),
						elem.Anchor(vecty.Markup(prop.Href("#")), vecty.Text("EIEIO")),
					),
					elem.ListItem(
						elem.Anchor(vecty.Markup(prop.Href("https://godoc.org/github.com/spatialmodel/inmap")), vecty.Text("API")),
					),
					elem.ListItem(
						elem.Anchor(vecty.Markup(prop.Href("/help")), vecty.Text("Help")),
					),
					elem.ListItem(
						elem.Anchor(vecty.Markup(prop.Href("/blog")), vecty.Text("Blog")),
					),
				),
			),
		),
	)
}
