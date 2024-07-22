import {add} from './lib/css.js';
import {br, div, h2, h3, li, main, pre, samp, ul} from './lib/html.js';

const base = main({"id": "about"}, [
	h2("About"),
	div([
		"SoftPack is a software packaging platform that creates stable software environments that can be discovered and shared with others.",
		br(),
		br(),
		"These multi-user, cross-platform environments can consist of any software you like and aid in reproducable research by ensuring you can always return to the exact set of software (the same versions, including the same versions of all dependencies) as you used previously."
	]),
	h3("Discovering Environments"),
	div([
		"Before creating a new software environment for yourself, you can find if someone has already made an environment that you can use.",
		br(),
		br(),
		`Click on the "Environments" link in the sidebar to the left. This will show you a list of existing environments. Ones with a green "ready" bar on their left can be used. Click on it for instructions on how to use it.`,
		br(),
		br(),
		`You can also search for environments by entering one or more search terms in the "Search for Environments" box at the top of the Environments screen. You can enter names of software packages or the name of an environment. You can also filter by the name of the user or group environments were created for, and by tag to get environments in certain categories. Finally, you can filter by your own environemnts by entering your username in the top right "Username" field and selecting "mine", and for environements that are currently building by ticking "building".`
	]),
	h3("Using Environments"),
	div([
		"The environments that SoftPack creates are actually singularity images, which provide the reproducability. To make these easy to use on the farm, SoftPack wraps these images with modules and wrapper scripts, so that after loading the module, you can use the software in the environment as if it was installed locally.",
		br(),
		br(),
		"First you need to enable the module system and tell it where to look for environments. To do this, you should have something like the following in your ~/.bashrc file:",
		pre(`source /etc/profile.d/modules.sh
export MODULEPATH=$MODULEPATH:/software/modules/`),
		"Now you can copy the ",
		samp("module load"),
		" command in the Usage instructions for your discovered environment, and paste it in to your terminal, where you've ssh'd to a farm node. The Description of your environment will tell you what new executables will now be in your $PATH.",
		br(),
		br(),
		"For example, if you're using an environment with R modules in it, running R from your terminal will now use the R in the singularity container where the desired R modules are available to it.",
	]),
	h3("RStudio"),
	div([
		"When using a SoftPack environment with R modules in it, you may wish to use those R modules in RStudio. Rather than create an environment that has the R modules and RStudio in it, we recommend excluding RStudio from the environment, and instead use our separate RStudio module on top.",
		br(),
		br(),
		"First ",
		samp("module load"),
		" your SoftPack environment as normal, then ",
		samp("module load HGI/common/rstudio"),
		". You can then run ",
		samp("rstudio start"),
		" to launch an instance of rstudio on the farm that will have access to the R modules in your SoftPack environment. ",
		samp("rstudio stop"),
		" when you're done. See ",
		samp("rstudio --help"),
		" for more information, and in particular ",
		samp("rstudio start --help"),
		" for how to use your own locally installed R modules with it as well."
	]),
	h3("Creation"),
	`You can create environments from scratch, or based on a pre-existing environment ("clone"). To clone an environment, discover it in the usual way, click on it and then click the "clone" button in the top right of the information panel that appears. This will fill out the "Environment Settings" form described below, and you can alter the fields as desired.`,
	br(),
	br(),
	"If recipes already exist for all your desired software, you'll be able to use the web frontend to create software environments for yourself. If recipes don't exist, you'll have to contact the admin and have them create the recipe first.",
	br(),
	br(),
	`Following is an example of creating an environment of your own for the "xxhash" software.`,
	ul([
		li(`Click the "Create Environment" link in the left-hand side-bar.`),
		li(`If you haven't already, enter your username in the top right "Username" field.`),
		li(`Enter a name for this environment, a description, optional tags and then click on the folder dropdown and select users/[your username] or a group to indicate the environment is useful to other members of that group.`),
		li(`In the "Package Settings" section, start typing "xxhash" in to the "Packages" field. A selection of matching packages will pop up; select the "xxhash" result.`),
		li(`Without changing anything else you'd install the latest version (that there is a spack recipe for). Instead, click the little arrow in the xxhash entry and select 0.7.4 to install an older version.`),
		li(`Now click the "CREATE" button, which should result in a pop-up message saying your request has been scheduled. (If you see a "Environment build failed" message instead, contact the admin to investigate.)`)
	]),
	"The creation of your environment will take an unknown amount of time, depending on how many other builds have been scheduled before yours. It could be minutes or an entire day; please be patient.",
	br(),
	br(),
	`You will have been taken to the "Environments" page and it will show your building environments. Once it finishes building it will disappear from that view, so uncheck "building" to find it.`,
	br(),
	br(),
	`If it turns red to indicate a failure, contact your admin and they'll investigate. Otherwise, you can click it to get its module load command similar to the "Usage" section above.`,
	br(),
	br(),
	`If you no longer want this tutorial environment, please contact the admin and we'll delete it for you (deletion isn't currently available from the frontend).`
]);

add("#about", {
	"max-width": "40em",
	"color": "rgba(0, 0, 0, 0.87)",
	"font-weight": 400,
	"line-height": 1.5,
	"font-size": "0.85em",

	" h2": {
		"font-size": "2em",
	},

	" pre": {
		"border": "1px solid #888",
		"padding": "0.5em",
		"background-color": "#eee"
	},
	" samp": {
		"border": "1px solid #888",
		"padding": "2px",
		"margin": "1px",
		"background-color": "#eee"
	}
});

export default () => base;
