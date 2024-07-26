import {add, render} from './lib/css.js';
import {amendNode} from './lib/dom.js';
import {a, h1, header, li, menu, nav} from './lib/html.js';
import ready from './lib/load.js';
import {router} from './lib/router.js';
import {rpcInit} from './rpc.js';
import About from './about.js';
import Environments, {ready as environmentsReady} from './environments.js';
import EnvironmentDrawer from './environmentDrawer.js';
import Tags from './tags.js';
import Create from './create.js';
import UserInput from './users.js';

ready.then(() => rpcInit("envs/socket"))
.then(() => environmentsReady)
.then(() => {
	amendNode(document.head, render());
	amendNode(document.body, [
		header([
			h1("SoftPack"),
			UserInput
		]),
		nav(menu([
			li(a({"href": "about"}, "About")),
			li(a({"href": "environments"}, "Environments")),
			li(a({"href": "tags"}, "Tags")),
			li(a({"href": "create"}, "Create Environment"))
		])),
		router()
		.add("about", About)
		.add("environments?envId=:path", Environments)
		.add("tags", Tags)
		.add("create", Create)
		.add("", Environments),
		router().add("?envId=:path", EnvironmentDrawer)
	]);
});

add({
	"body": {
		"padding": 0,
		"margin": 0,
		"font-family": `"Inter", -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif, "Apple Color Emoji", "Segoe UI Emoji"`,
		"color": "rgba(0, 0, 0, 0.87)",
		"font-weight": 400,
		"font-size": 14
	},
	"header": {
		"position": "fixed",
		"z-index": 10,
		"background-color": "#fff",
		"width": "100%",
		"height": "64px",
		"top": 0,
		"left": 0,
		"align-items": "center",
		"display": "flex",
		"padding": 0,
		"box-shadow": "rgba(159, 162, 191, 0.18) 0px 9px 16px, rgba(159, 162, 191, 0.32) 0px 2px 2px",

		" h1": {
			"font-size": "16px",
			"padding-left": "24px",
			"flex-grow": 1
		}
	},
	"nav": {
		"position": "fixed",
		"width": "240px",
		"border-right": "1px solid rgba(0, 0, 0, 0.12)",
		"top": "64px",
		"left": 0,
		"bottom": 0,

		" menu": {
			"list-style": "none",
			"padding": 0,
			"margin": 0,

			" li": {
				"background-repeat": "no-repeat",
				"background-size": "1.5em 1.5em",
				"background-position": "1em 0.7em",

				" a": {
					"display": "block",
					"align-items": "center",
					"padding": "14px 16px 14px 72px",
					"font-size": "14px",
					"text-decoration": "none",
					"color": "rgba(0, 0, 0, 0.87)",
					"transition": "background-color 150ms cubic-bezier(0.4, 0, 0.2, 1)",

					":hover": {
						"background-color": "rgba(0, 0, 0, 0.04)"
					}
				},
				
				":nth-child(1)": {
					"background-image": `url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' %3E%3Cpath fill-opacity='0.54' d='M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2m-6.99 15c-.7 0-1.26-.56-1.26-1.26 0-.71.56-1.25 1.26-1.25.71 0 1.25.54 1.25 1.25-.01.69-.54 1.26-1.25 1.26m3.01-7.4c-.76 1.11-1.48 1.46-1.87 2.17-.16.29-.22.48-.22 1.41h-1.82c0-.49-.08-1.29.31-1.98.49-.87 1.42-1.39 1.96-2.16.57-.81.25-2.33-1.37-2.33-1.06 0-1.58.8-1.8 1.48l-1.65-.7C9.01 7.15 10.22 6 11.99 6c1.48 0 2.49.67 3.01 1.52.44.72.7 2.07.02 3.08'%3E%3C/path%3E%3C/svg%3E")`
				},

				":nth-child(2)": {
					"background-image": `url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' %3E%3Cpath fill-opacity='0.54' d='M13 13v8h8v-8zM3 21h8v-8H3zM3 3v8h8V3zm13.66-1.31L11 7.34 16.66 13l5.66-5.66z'%3E%3C/path%3E%3C/svg%3E")`
				},

				":nth-child(3)": {
					"background-image": `url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' %3E%3Cpath fill-opacity='0.54' d='m21.41 11.58-9-9C12.05 2.22 11.55 2 11 2H4c-1.1 0-2 .9-2 2v7c0 .55.22 1.05.59 1.42l9 9c.36.36.86.58 1.41.58.55 0 1.05-.22 1.41-.59l7-7c.37-.36.59-.86.59-1.41 0-.55-.23-1.06-.59-1.42M5.5 7C4.67 7 4 6.33 4 5.5S4.67 4 5.5 4 7 4.67 7 5.5 6.33 7 5.5 7'%3E%3C/path%3E%3C/svg%3E")`
				},

				":nth-child(4)": {
					"background-image": `url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24'%3E%3Cpath fill-opacity='0.54' d='m13.7826 15.1719 2.1213-2.1213 5.9963 5.9962-2.1213 2.1213zM17.5 10c1.93 0 3.5-1.57 3.5-3.5 0-.58-.16-1.12-.41-1.6l-2.7 2.7-1.49-1.49 2.7-2.7c-.48-.25-1.02-.41-1.6-.41C15.57 3 14 4.57 14 6.5c0 .41.08.8.21 1.16l-1.85 1.85-1.78-1.78.71-.71-1.41-1.41L12 3.49c-1.17-1.17-3.07-1.17-4.24 0L4.22 7.03l1.41 1.41H2.81l-.71.71 3.54 3.54.71-.71V9.15l1.41 1.41.71-.71 1.78 1.78-7.41 7.41 2.12 2.12L16.34 9.79c.36.13.75.21 1.16.21'%3E%3C/path%3E%3C/svg%3E")`
				}
			},
		}
	},
	"main": {
		"margin-left": "255px",
		"margin-top": "85px",
		"margin-right": "2em"
	}
});
