// This script is executed in the browser console at https://golangci-lint.run/usage/linters
//
// This script is used to print a direct listing of the golangci-lint linters to
// compare with the linters in my configuration.
let linterList = [];

let linterListDeprecated = [
	"- varcheck",
	"- structcheck",
	"- deadcode",
	"- nosnakecase",
	"- exhaustivestruct",
	"- golint",
	"- ifshort",
	"- interfacer",
	"- maligned",
	"- scopelint",
	"- gomnd",
	"- execinquery",
];

let myBlacklist = [
	"- gofmt",
	"- gofumpt",
	"- goimports",
	"- lll",
	"- gci",
	"- nonamedreturns",
	"- exhaustruct",
	"- musttag",
	"- tagliatelle",
	"- tagalign",
	"- godox",
	"- inamedparam",
	"- gochecksumtype",
	"- depguard",
];

document.querySelectorAll("table > tbody").forEach((tableBody) => {
	tableBody.childNodes.forEach((tr) => {
		let linterName = tr.querySelector("td").innerText;

		linterName = linterName.split(" ")[0].split(" ")[0].split(" ")[0];

		linterName = "- " + linterName;

		if (
			myBlacklist.includes(linterName) ||
			linterListDeprecated.includes(linterName)
		) {
			return;
		}

		let linterDescription = tr.querySelectorAll("td")[1].innerText;

		if (
			linterDescription.includes("Replaced") ||
			linterDescription.includes("deprecated") ||
			linterDescription.includes("archived")
		) {
			linterListDeprecated.push(linterName);
		} else {
			linterList.push(linterName);
		}
	});
});

let uniqueLinterList = [...new Set(linterList)];
let uniqueLinterListDeprecated = [...new Set(linterListDeprecated)];

clear();

console.log("linters.enable:");
console.log(uniqueLinterList.join("\n"));

console.log("");

console.log("linters.disable:");
console.log(uniqueLinterListDeprecated.join("\n"));
console.log("");
console.log(myBlacklist.join("\n"));
