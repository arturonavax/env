require("lvim.lsp.null-ls.formatters").setup({
	-- shell
	{ command = "beautysh" },
	{
		command = "shfmt",
		filetypes = { "sh", "bash", "zsh" },
	},
	{
		command = "shellharden",
		filetypes = { "sh", "bash" },
	},
	-- c/c++
	{
		command = "clang-format",
		extra_args = { "--style", "{UseTab: 'Always', IndentWidth: 4, TabWidth: 4}" },
	},
	-- css
	{ command = "stylelint" },
	-- python
	{ command = "black" },
	-- protobuf
	{ command = "buf" },
	-- just
	{ command = "just" },
	-- lua
	{ command = "stylua" },
	-- nginx
	{ command = "nginx_beautifier" },
	-- sql
	{
		command = "sql-formatter",
		args = {
			"--config",
			os.getenv("HOME") .. "/.sql-formatter.json",
		},
	},
	-- others
	{
		command = "prettier",
		filetypes = {
			"vue",
			"css",
			"scss",
			"less",
			"html",
			"json",
			"jsonc",
			"javascript",
			"typescript",
			"javascriptreact",
			"typescriptreact",
			"yaml",
			"markdown",
			"markdown.mdx",
			"graphql",
			"handlebars",
		},
		-- args config default
		args = {
			"--config-precedence",
			"file-override",
			"--tab-width",
			"4",
			"--use-tabs",
			"--print-width",
			"80",
			"--arrow-parens",
			"always",
			"--quote-props",
			"as-needed",
			"--trailing-comma",
			"all",
		},
	},
})
