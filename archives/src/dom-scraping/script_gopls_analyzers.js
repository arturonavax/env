// This script is executed in the browser console at https://github.com/golang/tools/blob/master/gopls/doc/analyzers.md
let all = "";

document.querySelectorAll("h2>strong").forEach((e) => {
	if (e.textContent == "analyzers.md") return;

	all += `"${e.textContent}": true,\n`;
});

console.log(all);
