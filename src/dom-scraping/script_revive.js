// This script is executed in the browser console at https://github.com/mgechev/revive#available-rules
//
// This script is used to print a direct listing of the revive linters to
// compare with the linters in my configuration.
let linterList = [];

let myBlacklist = [
	"add-constant",
	"package-comments",
	"cyclomatic",
	"max-public-structs",
	"file-header",
	"unhandled-error",
	"cognitive-complexity",
	"string-format",
	"function-length",
	"nested-structs",
	"banned-characters",
];

document.querySelector("table > tbody").childNodes.forEach((tr) => {
	if (tr.nodeName != "TR") {
		return;
	}

	let linterName = tr.querySelector("td").innerText;

	linterName = linterName.split(" ")[0].split(" ")[0].split(" ")[0];

	if (myBlacklist.includes(linterName)) {
		return;
	}

	linterName = "- name: " + linterName;

	linterList.push(linterName);
});

let uniqueLinterList = [...new Set(linterList)];

clear();

console.log("linters-settings.revive.rules:");
console.log(uniqueLinterList.join("\n"));
