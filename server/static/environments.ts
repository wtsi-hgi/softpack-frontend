import type {Binding} from './lib/bind.js';
import type {MultiSelect} from './lib/multiselect.js';
import type {Subscribed} from './lib/inter.js';
import bind from './lib/bind.js';
import {add} from './lib/css.js';
import {amendNode, bindCustomElement} from './lib/dom.js';
import {div, h2, input, label, li, main, span, ul} from './lib/html.js';
import {debounce} from './lib/misc.js';
import {multioption, multiselect} from './lib/multiselect.js';
import {setAndReturn} from './lib/misc.js';
import {NodeMap, node, stringSort} from './lib/nodes.js';
import {goto} from './lib/router.js';
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
	tags: Binding<string[]>;
	#state: Binding<number>;
	user = "";
	group = "";
	packages: Binding<[string, string?][]>;
	name: string;
	version: string;
	readme: Binding<string>;
	description: Binding<string>;

	constructor(path: string, envData: NonNullable<Subscribed<typeof environmentUpdate>[""]>) {
		const pathParts = path.split("/"),
		      nameVer = pathParts.pop()!.split("-"),
		      version = nameVer.length > 1 ? nameVer.pop()! : "",
		      name = nameVer.join("-"),
		      noVersion = pathParts.join("/") + "/" + name,
		      otherVersions = versionMap.get(noVersion) ?? setAndReturn(versionMap, noVersion, new Versions(filter));

		if (pathParts[0] === "users") {
			users.addEntry(this.user = pathParts[1]);
		} else if (pathParts[0] === "groups") {
			groups.addEntry(this.group = pathParts[1]);
		}

		this[node] = li({"class": `${envData.SoftPack ? "softpack" : "module"} ${(this.#state = bind(envData.Status)).transform(state => statuses[state])}`, "onclick": () => {
			goto(`?envId=${encodeURIComponent(path)}`);
		}}, [
			h2(`${this.name = name}${(this.version = version)  ? "-" + version : ""}`),
			ul({"class": "pathParts"}, pathParts.map(part => li(part))),
			(this.tags = bind(envData.Tags)).toDOM(ul({"class": "tags"}), tag => li(tag)),
			div((this.description = bind(envData.Description)).transform(description => description.split("\n")[0])),
			(this.packages = bind(envData.Packages.map(pkg => pkg.toLowerCase().split("@", 2) as [string, string?]))).toDOM(ul({"class": "packages"}), pkg => li(pkg[0] + (pkg[1] ? "@" + pkg[1] : "")))
		]);
		this.#sortKey = Array.from(version.matchAll(/\d+/g)).reduce((s, [p]) => s + p.padStart(20, "0"), name) + version.replaceAll(/\d/g, "") + pathParts.join("/");
		this.readme = bind(envData.ReadMe);

		otherVersions.add(this);

		for (const tag of envData.Tags) {
			tags.addEntry(tag);
		}
	}

	update(envData: NonNullable<Subscribed<typeof environmentUpdate>[""]>) {
		for (const tag of this.tags()) {
			tags.removeEntry(tag);
		}

		for (const tag of envData.Tags) {
			tags.addEntry(tag);
		}

		this.tags(envData.Tags);
		this.description(envData.Description);
		this.packages(envData.Packages.map(pkg => pkg.toLowerCase().split("@", 2) as [string, string?]));
		this.readme(envData.ReadMe);
	}

	compare(b: Environment) {
		return stringSort(this.#sortKey, b.#sortKey);
	}

	#filter(filter: Filter) {
		if (filter.building) {
			if (this.#state() === 2) {
				return false;
			}
		} else if (this.#state() === 1) {
			return false;
		}

		if (filter.tags.length > 0 && !filter.tags.every(tag => this.tags().includes(tag))) {
			return false;
		}

		if ((filter.groups.length > 0 || filter.users.length > 0) && !filter.groups.includes(this.group) && filter.users.length > 0 && !filter.users.includes(this.user)) {
			return false;
		}

		const packages = this.packages();

		return filter.terms.length === 0 || filter.terms.every(([name, ver]) => {
			if (name && this.name.includes(name)) {
				return true;
			}

			if (ver && this.version.startsWith(ver)) {
				return true;
			}

			for (const [pkgName, pkgVer] of packages) {
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
		for (const tag of this.tags()) {
			tags.removeEntry(tag);
		}

		if (this.user) {
			users.removeEntry(this.user);
		} else {
			groups.removeEntry(this.group);
		}
	}
}

class EnvironmentList extends HTMLElement {
	#element: Element;
	#filter: Filter;
	#debounce = debounce();

	constructor(child: Element, filter: Filter) {
		super();

		this.#filter = filter;
		this.append(this.#element = child);
	}

	filter(filter: Filter) {
		this.#filter = filter;

		this.#debounce(() => {
			for (const envs of versionMap.values()) {
				envs.filter(filter);
			}
		});
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

class FilterList {
	#counts = new Map<string, {count: number}>();
	#debounce = debounce();
	values = bind([] as string[]);

	addEntry(name: string) {
		const entry = this.#counts.get(name) ?? setAndReturn(this.#counts, name, {count: 0});

		entry.count++;

		this.#update();
	}

	removeEntry(name: string) {
		const entry = this.#counts.get(name) ?? {count: 1};

		entry.count--;

		if (entry.count <= 0) {
			this.#counts.delete(name);
		}

		this.#update();
	}

	#update() {
		this.#debounce(() => {
			this.values(Array.from(this.#counts.keys()).sort(stringSort));
		});
	}

	has(key: string) {
		return this.#counts.has(key);
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
      users = new FilterList(),
      groups = new FilterList(),
      tags = new FilterList(),
      versionMap = new Map<string, Versions>(),
      envSorter = (a: Environment, b: Environment) => a.compare(b),
      environments = new NodeMap<string, Environment, HTMLUListElement>(ul({"id": "environments"}), envSorter),
      environmentFilter = environmentContainer(),
      userFilter = users.values.toDOM(multiselect({"placeholder": "Filter by user", "onchange": function (this: MultiSelect) {
		if (!this.value.includes(username())) {
			mine.checked = false;
		}
	}}), user => multioption(user)),
      groupFilter = groups.values.toDOM(multiselect({"placeholder": "Filter by group", "onchange": function (this: MultiSelect) {
		const selected = this.value;

		for (const group of groupList()) {
			if (!selected.includes(group) && groups.has(group)) {
				mine.checked = false;

				break;
			}
		}
      }}), group => multioption(group)),
      mine = input({"type": "checkbox", "id": "showMine", "checked": username.transform(() => false), "onclick": function (this: HTMLInputElement) {
	if (this.checked) {
		userFilter.value = [username()];
		groupFilter.value = groupList();
	} else {
		userFilter.value = [];
		groupFilter.value = [];
	}
      }}),
      base = main([
	input({"type": "search", "id": "filter", "placeholder": "Search for environments by name of package[@version]", "oninput": function (this: HTMLInputElement) {
		filter.terms = this.value.trim().split(/\s+/g).filter(t => t).map(p => p.split("@", 2) as [string, string?]);

		environmentFilter.filter(filter);
	}}),
	userFilter,
	groupFilter,
	tags.values.toDOM(multiselect({"placeholder": "Filter by tag"}), tag => multioption(tag)),
	input({"type": "checkbox", "id": "showBuilding", "onclick": function (this: HTMLInputElement) {
		environmentFilter.setShowBuilding(this.checked);
	}}),
	label({"for": "showBuilding"}, "Building"),
	input({"type": "checkbox", "id": "showAllVersions", "onclick": function (this: HTMLInputElement) {
		environmentFilter.setShowAllVersion(this.checked);
	}}),
	label({"for": "showAllVersions"}, "All Versions"),
	span({"style": groupList.transform(list => list.length ? "" : "display: none")}, [
		mine,
		label({"for": "showMine"}, "Mine"),
	]),
	environmentFilter
      ]),
      {promise: ready, resolve: firstLoad} = Promise.withResolvers<void>();

environmentUpdate.when(envs => {
	for (const [path, envData] of Object.entries(envs)) {
		const existing = environments.get(path)

		if (envData) {
			if (existing) {
				existing.update(envData);
			} else {
				environments.set(path, new Environment(path, envData));
			}
		} else if (existing) {
			existing.cleanup();
			environments.delete(path);
		}
	}

	firstLoad();
});

export {ready, environments};

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
			}
		}
	},

	"ul": {
		"list-style": "none",
		"padding": 0,

		".pathParts": {
			"display": "inline-block",
			"margin-left": "1em",

			" li:not(:first-child):before": {
				"content": `"›"`,
				"padding": "0 0.5em"
			}
		},

		">li": {
			"display": "inline-block"
		},

		".tags:not(:empty)": {
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

		".packages": {
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
});
