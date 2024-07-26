import {add} from './lib/css.js';
import {amendNode, bindCustomElement} from './lib/dom.js';
import {code, div, h2, hr, li, pre, ul} from './lib/html.js';
import Markdown from './lib/markdown.js';
import {goto} from './lib/router.js';
import {environments} from './environments.js';

class Opener extends HTMLDialogElement {
	connectedCallback() {
		amendNode(document.documentElement, {"class": {"modelOpen": true}});
		this.showModal();
	}

	disconnectedCallback() {
		amendNode(document.documentElement, {"class": {"modelOpen": false}});
		this.close();
	}
}

const dialog = bindCustomElement("dialog-opener", Opener, {"extends": "dialog"}),
      codeHandler = {
	      "code": (_: string, text: string) => pre({"class": "code"}, [
		div({"title": "Copy", "onclick": () => (navigator.clipboard ? navigator.clipboard.writeText(text) : Promise.reject())
		.catch(() => {
			const i = document.body.appendChild(div({"style": "position: fixed; top: 0; left: 0"}, text)),
			      range = document.createRange(),
			      selection = document.getSelection();

			range.selectNodeContents(i);
			selection?.removeAllRanges();
			selection?.addRange(range);
			document.execCommand("copy");
			i.remove();
		})}),
		code(text),
	      ])
      };

export default ({path}: {path?: string}) => {
	const env = environments.get(path ?? "");

	if (!env) {
		return new Text();
	}

	return dialog({"onclick": function(this: Opener, e: MouseEvent) {
		const {left} = this.getBoundingClientRect();

		if (e.clientX < left) {
			goto("?");
		}
	}}, [
		h2(env.name + (env.version ? "-" + env.version : "")),
		ul({"class": "pathParts"}, [
			li(env.group ? "groups" : "users"),
			li(env.group || env.user)
		]),
		hr(),
		Markdown(env.readme, codeHandler),
		hr(),
		h2("Description"),
		div({"class": "description"}, env.description),
		hr(),
		h2("Packages"),
		ul({"class": "packages"}, env.packages.map(pkg => li(pkg[0] + (pkg[1] ? "@" + pkg[1] : ""))))
	]);
};

add({
	"dialog": {
		"font-size": "14px",
		"margin": "0 0 0 auto",
		"padding": "40px",
		"height": "100%",
		"max-height": "100%",
		"box-sizing": "border-box",
		"width": "40em",
		"border": "none",
		"z-index": 1000,
		"overflow": "auto",
		"outline": "none",

		" h2:first-child": {
			"text-align": "center",
			"margin": 0,
		},

		" .pathParts": {
			"display": "flex",
			"justify-content": "center",
			"font-size": "14px",
			"font-family": `"Inter", -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif, "Apple Color Emoji", "Segoe UI Emoji"`,
			"font-weight": 400,
			"line-height": 1.5,
			"color": "rgba(0, 0, 0, 0.6)",
			"margin": 0
		},

		" .code": {
			"position": "relative",
			"color": "#fff",
			"background-color": "rgb(77, 64, 51)",
			"border": "0.3em solid rgb(122, 102, 82)",
			"border-radius": "0.5em",
			"box-shadow": "black 1px 1px 0.5em inset",
			"padding": "1em",
			"margin": "0.5em 0",
			"font-weight": 400,

			">code": {
				"text-shadow": "black 0px -0.1em 0.2em",
				"line-height": 1.5,
				"tab-size": 4,
			},

			">div:first-child": {
				"background-image": `url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24'%3E%3Cpath d='M16 1H4c-1.1 0-2 .9-2 2v14h2V3h12zm3 4H8c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h11c1.1 0 2-.9 2-2V7c0-1.1-.9-2-2-2m0 16H8V7h11z' fill='%23fff'%3E%3C/path%3E%3C/svg%3E")`,
				"background-repeat": "no-repeat",
				"background-size": "1.5em 1.5em",
				"width": "1.5em",
				"height": "1.5em",
				"position": "absolute",
				"top": 0,
				"right": 0,
				"cursor": "pointer",
			}
		},

		" .description": {
			"font-size": "14px",
			"font-family": `"Inter", -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif, "Apple Color Emoji", "Segoe UI Emoji"`,
			"font-weight": 400,
			"line-height": 1.5,
			"white-space": "pre-wrap"
		}
	},

	"::backdrop": {
		"background-color": "#000",
		"opacity": 0.5,
		"z-index": 999,
		"overscroll-behavior": "none"
	},

	"html": {
		"scrollbar-gutter": "stable",

		".modelOpen": {
			"overflow-y": "hidden"
		}
	}
});
