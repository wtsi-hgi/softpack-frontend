import type {MultiSelect, MultiOption} from './lib/multiselect.js';
import type {Subscribed} from './lib/inter.js';
import {add} from './lib/css.js';
import {amendNode, bindCustomElement} from './lib/dom.js';
import {div, h2, input, label, li, main, span, ul} from './lib/html.js';
import {multioption, multiselect} from './lib/multiselect.js';
import {setAndReturn} from './lib/misc.js';
import {NodeMap, node, stringSort} from './lib/nodes.js';
import {environmentUpdate} from './rpc.js';
import {groupList, username} from './users.js';

type Filter = {
	terms: [string, string?][];
	users: string[];
	groups: string[];
	tags: string[];
	building: boolean;
}

class Environment {
	[node]: HTMLLIElement;
	#sortKey: string;
	#tags: string[];
	#state: number;
	#user = "";
	#group = "";
	#packages: [string, string?][];
	#name: string;
	#version: string;

	constructor(path: string, envData: NonNullable<Subscribed<typeof environmentUpdate>[""]>){
		const pathParts = path.split("/"),
		      nameVer = pathParts.pop()!.split("-"),
		      version = nameVer.length > 1 ? nameVer.pop()! : "",
		      name = nameVer.join("-"),
		      noVersion = pathParts.join("/") + "/" + name,
		      otherVersions = versionMap.get(noVersion) ?? setAndReturn(versionMap, noVersion, new Versions(filter));

		for (const tag of this.#tags = envData.Tags) {
			tags.addEntry(tag);
		}

		if (pathParts[0] === "users") {
			users.addEntry(this.#user = pathParts[1]);
		} else if (pathParts[0] === "groups") {
			groups.addEntry(this.#group = pathParts[1]);
		}

		this[node] = li({"class": `${envData.SoftPack ? "softpack" : "module"} ${statuses[envData.Status]}`}, [
			h2(name + (version ? "-" + version : "")),
			ul(pathParts.map(part => li(part))),
			ul(envData.Tags.map(tag => li(tag))),
			div(envData.Description.split("\n")[0]),
			ul(envData.Packages.map(pkg => li(pkg)))
		]);
		this.#name = name;
		this.#version = version;
		this.#state = envData.Status;
		this.#packages = envData.Packages.map(pkg => pkg.toLowerCase().split("@", 2) as [string, string?]);
		this.#sortKey = Array.from(version.matchAll(/\d+/g)).reduce((s, [p]) => s + p.padStart(20, "0"), name) + version.replaceAll(/\d/g, "")+pathParts.join("/");

		otherVersions.add(this);
	}

	compare(b: Environment) {
		return stringSort(this.#sortKey, b.#sortKey);
	}

	#filter(filter: Filter) {
		if (filter.building) {
			if (this.#state === 2) {
				return false;
			}
		} else if (this.#state === 1) {
			return false;
		}

		if (filter.tags.length > 0 && !filter.tags.every(tag => this.#tags.includes(tag))) {
			return false;
		}

		if ((filter.groups.length > 0 || filter.users.length > 0) && !filter.groups.includes(this.#group) && filter.users.length > 0 && !filter.users.includes(this.#user)) {
			return false;
		}

		return filter.terms.length === 0 || filter.terms.every(([name, ver]) => {
			if (name && this.#name.includes(name)) {
				return true;
			}

			if (ver && this.#version.startsWith(ver)) {
				return true;
			}

			for (const [pkgName, pkgVer] of this.#packages) {
				if (name && pkgName.includes(name)) {
					return true;
				}

				if (ver && pkgVer?.startsWith(ver)) {
					return true;
				}
			}

			return false;
		});
	}

	filter(filter: Filter) {
		const match = this.#filter(filter);

		if (match) {
			amendNode(this, filterMatch);
		} else {
			amendNode(this, filterUnmatch);
		}

		return match;
	}

	setLatest() {
		amendNode(this, latestVersion);
	}

	setOldVersion() {
		amendNode(this, olderVersion);
	}

	cleanup() {
		for (const tag of this.#tags) {
			tags.removeEntry(tag);
		}

		if (this.#user) {
			users.removeEntry(this.#user);
		} else {
			groups.removeEntry(this.#group);
		}
	}
}

class EnvironmentList extends HTMLElement {
	#element: Element;
	#filter: Filter;
	#debounce = false;

	constructor(child: Element, filter: Filter) {
		super();

		this.#filter = filter;
		this.append(this.#element = child);
	}

	filter(filter: Filter) {
		this.#filter = filter;

		if (!this.#debounce) {
			this.#debounce = true;

			setTimeout(() => {
				for (const envs of versionMap.values()) {
					envs.filter(filter);
				}

				this.#debounce = false;
			});
		}
	}

	setShowAllVersion(v: boolean) {
		amendNode(this, v ? showAllVersions : showLatestVersion);
	}

	setShowBuilding(v: boolean) {
		this.#filter.building = v;

		this.filter(this.#filter);
	}

	connectedCallback() {
		if (this.#element.parentNode === this) {
			return;
		}

		if (this.#filter) {
			this.filter(this.#filter);
		}

		this.replaceChildren(this.#element);
	}
}

class Versions {
	#versions: Environment[] = [];
	#filter: Filter;

	constructor(filter: Filter) {
		this.#filter = filter;
	}

	add(e: Environment) {
		this.#versions.push(e);
		this.#versions.sort(envSorter).reverse();

		this.filter(this.#filter);
	}

	filter(filter: Filter) {
		this.#filter = filter;

		let needLatest = true;

		for (const env of this.#versions) {
			if (env.filter(filter)) {
				if (needLatest) {
					env.setLatest();

					needLatest = false;
				} else {
					env.setOldVersion();
				}
			}
		}
	}
}

class FilterList extends NodeMap<string, {[node]: MultiOption, name: string, count: number}, MultiSelect> {
	constructor(name: Exclude<keyof Filter, "terms" | "building">) {
		super(multiselect(), (a, b) => stringSort(a.name, b.name));

		amendNode(this, {"onchange": function(this: MultiSelect) {
			filter[name] = this.value as string[];

			environmentFilter.filter(filter);
		}});
	}

	addEntry(name: string) {
		const entry = this.get(name) ?? setAndReturn(this, name, {[node]: multioption(name), name, count: 0});

		if (!entry.count) {
		}

		entry.count++;
	}

	removeEntry(name: string) {
		const entry = this.get(name) ?? {count: 1};

		entry.count--;

		if (entry.count <= 0) {
			this.delete(name);
		}
	}
}

bindCustomElement("environment-list", EnvironmentList);

export const environmentContainer = () => new EnvironmentList(environments[node], filter);

const statuses = ["building", "failed", "ready"],
      filterMatch = {"class": {"filtered": false}},
      filterUnmatch = {"class": {"filtered": true}},
      latestVersion = {"class": {"oldVersion": false}},
      olderVersion = {"class": {"oldVersion": true}},
      showAllVersions = {"class": {"allVersions": true}},
      showLatestVersion = {"class": {"allVersions": false}},
      filter: Filter = {"terms": [], "tags": [], "groups": [], "users": [], "building": false},
      users = new FilterList("users"),
      groups = new FilterList("groups"),
      tags = new FilterList("tags"),
      versionMap = new Map<string, Versions>(),
      envSorter = (a: Environment, b: Environment) => a.compare(b),
      environments = new NodeMap<string, Environment, HTMLUListElement>(ul({"id": "environments"}), envSorter),
      environmentFilter = environmentContainer(),
      mine = input({"type": "checkbox", "id": "showMine", "checked": username.transform(() => false), "onclick": function(this: HTMLInputElement) {
	if (this.checked) {
		users[node].value = [username()];
		groups[node].value = groupList();
	} else {
		users[node].value = [];
		groups[node].value = [];
	}
      }}),
      base = main([
	input({"type": "search", "id": "filter", "placeholder": "Search for environments by name of package[@version]", "oninput": function(this: HTMLInputElement) {
		filter.terms = this.value.trim().split(/\s+/g).filter(t => t).map(p => p.split("@", 2) as [string, string?]);

		environmentFilter.filter(filter);
	}}),
	amendNode(users, {"placeholder": "Filter by user", "onchange": function(this: MultiSelect) {
		if (!this.value.includes(username())) {
			mine.checked = false;
		}
	}}),
	amendNode(groups, {"placeholder": "Filter by group", "onchange": function(this: MultiSelect) {
		const selected = this.value;

		for (const group of groupList()) {
			if (!selected.includes(group) && groups.has(group)) {
				mine.checked = false;

				break;
			}
		}
	}}),
	amendNode(tags, {"placeholder": "Filter by tag"}),
	input({"type": "checkbox", "id": "showBuilding", "onclick": function(this: HTMLInputElement) {
		environmentFilter.setShowBuilding(this.checked);
	}}),
	label({"for": "showBuilding"}, "Building"),
	input({"type": "checkbox", "id": "showAllVersions", "onclick": function(this: HTMLInputElement) {
		environmentFilter.setShowAllVersion(this.checked);
	}}),
	label({"for": "showAllVersions"}, "All Versions"),
	span({"style": groupList.transform(list => list.length ? "" : "display: none")}, [
		mine,
		label({"for": "showMine"}, "Mine"),
	]),
	environmentFilter
      ]);

environmentUpdate.when(envs => {
	for (const [path, envData] of Object.entries(envs)) {
		environments.get(path)?.cleanup();

		if (envData) {
			environments.set(path, new Environment(path, envData));
		} else {
			environments.delete(path);
		}
	}
});

export default () => base;

add({
	"#filter": {
		"width": "100%",
		"margin-bottom": "1em",
	},

	"multi-select": {
		"display": "inline-block",
		"margin": "0 1em",
		"width": "160px",
		"--removeXColor": "#ebebeb",
		"--removeBackgroundColor": "rgb(255, 94, 123)",
		"--removeBorderColor": "rgb(255, 94, 123)",
		"--selectedBackground": "#ebebeb",
		"--selectedBorderRadius": "16px",
		"--selectedPadding": "0 8px",
	},

	"label": {
		"margin": "0 1em",
	},

	"environment-list:not(.allVersions) #environments>li.oldVersion": {
		"display": "none"
	},

	"#environments": {
		"list-style": "none",
		"padding": 0,
		"display": "grid",
		"grid-template-columns": "repeat(auto-fit, minmax(40%, 1fr))",

		">li": {
			"position": "relative",
			"padding": "1em",
			"margin": "0 0 1em 1em",
			"background-color": "rgba(34, 51, 84, 0.02)",
			"box-shadow": "rgba(159, 162, 191, 0.18) 0px 9px 16px, rgba(159, 162, 191, 0.32) 0px 2px 2px",
			"border-radius": "10px",

			".filtered": {
				"display": "none"
			},

			":after": {
				"position": "absolute",
				"top": "0.1em",
				"right": "0.1em",
				"border": "1px solid #000",
				"border-radius": "50%",
				"width": "1.25em",
				"height": "1.25em",
				"text-align": "center"
			},

			".module:after": {
				"content": `"M"`
			},

			".softpack:after": {
				"content": `"S"`
			},

			":before": {
				"position": "absolute",
				"content": `" "`,
				"top": 0,
				"bottom": 0,
				"left": 0,
				"border-radius": "0.25em",
				"width": "0.5em"
			},

			".ready:before": {
				"background-color": "#57ca22"
			},

			".queued:before": {
				"background-color": "#33c2ff"
			},

			".failed:before": {
				"background-color": "#ff1943"
			},

			">h2": {
				"display": "inline",
				"margin": 0,
				"word-break": "break-all"
			},

			">div": {
				"margin": "1em 0"
			},

			">ul": {
				"list-style": "none",
				"padding": 0,

				":first-of-type": {
					"display": "inline-block",
					"margin-left": "1em",

					" li:not(:first-child):before": {
						"content": `"/"`,
						"padding": "0 0.5em"
					}
				},

				">li": {
					"display": "inline-block"
				},

				":nth-of-type(2):not(:empty)": {
					"margin-top": "1em",

					">li": {
						"color": "rgba(0, 0, 0, 0.87)",
						"background-color": "rgba(0, 0, 0, 0.08)",
						"font-size": "0.8125rem",
						"border-radius": "16px",
						"padding": "0.5em",
						"margin": "0 0 0.25em 0.25em"
					}
				},

				":nth-of-type(3)": {
					"max-height": "90px",
					"overflow-y": "auto",

					">li": {
						"font-size": "0.8125rem",
						"color": "rgba(85, 105, 255, 0.7)",
						"border": "1px solid rgba(85, 105, 255, 0.7)",
						"border-radius": "16px",
						"padding": "0.5em",
						"margin": "0 0 0.25em 0.25em"
					}
				}

			}
		}
	}
});
