import bind from './lib/bind.js';
import {add} from './lib/css.js';
import {div, input, label} from './lib/html.js';
import {getUserGroups} from './rpc.js';

export const username = bind(window.localStorage.getItem("username") ?? ""),
groupList = bind<string[]>([]);

if (username() != "") {
	getUserGroups(username()).then(groupList);
}

username.onChange(user => getUserGroups(user).then(groupList));

export default div({"id": "userInput"}, [
	label({"for": "userInput"}, "Username"),
	input({"type": "search", "id": "userInput", "value": username(), "onchange": function(this: HTMLInputElement) {window.localStorage.setItem("username", username(this.value));}})
]);

add("#userInput", {
	" label": {
		":after": {
			"content": `":"`
		}
	}
});
