require("lvim.lsp.null-ls.linters").setup({
	-- go
	require("go.null_ls").golangci_lint(), -- golangci-lint by go.nvim
	-- { command = "golangci-lint" }, -- golangci-lint by null-ls
	-- shell
	{ command = "shellcheck" },
	-- css
	{ command = "stylelint" },
	-- javascript/typescript
	{ command = "eslint" },
	-- python
	{ command = "pylint" },
	-- protobuf
	{ command = "buf" },
	-- zsh
	{ command = "zsh" },
	-- make
	{ command = "checkmake" },
	-- cmake
	{ command = "cmake_lint" },
	-- markdown
	{ command = "markdownlint" },
	-- github action
	{ command = "actionlint" },
})

-- disable shellcheck in *.env files
vim.cmd("autocmd BufNewFile,BufRead *.env setfiletype env")

-- css to scss
-- vim.cmd("autocmd BufNewFile,BufRead *.css setfiletype scss")

-- json to jsonc
-- vim.cmd("autocmd BufNewFile,BufRead *.json setfiletype jsonc")
