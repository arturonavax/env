require("lvim.lsp.null-ls.code_actions").setup({
	-- refactoring
	{ command = "refactoring" },
	-- shell
	{ command = "shellcheck" },
	-- javascript/typescript
	{ command = "eslint" },
	-- git blame
	{ command = "gitsigns" },
})
