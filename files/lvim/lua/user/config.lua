Root_patterns = {
	-- go
	"go.mod",
	"go.work",
	-- javascript/typescript
	"package.json",
	-- rust
	"Cargo.toml", -- cargo
	-- python
	"requirement.txt",
	-- java
	"pom.xml", -- apache maven
	-- php
	"composer.json", -- composer
	-- c++
	"CMakeLists.txt",
	-- swift
	"Package.swift",
	-- kotlin
	"build.gradle",
	-- version control systems (vcs)
	".git", -- git
	".hg", -- mercurial
	".bzr", -- bazaar
	".svn", -- subversion
	"_darcs", -- darcs
	-- generals
	"Makefile", -- make
	".null-ls-root",
	".root",
}

lvim.leader = "space"
lvim.log.level = "warn"
lvim.format_on_save = true
vim.opt.shell = "/bin/sh"
vim.opt.cmdheight = 1
vim.opt.completeopt = { "menuone", "noselect" }
-- vim.opt.foldmethod = "manual"
-- vim.opt.foldexpr = ""
vim.opt.foldmethod = "expr"
vim.opt.foldexpr = "nvim_treesitter#foldexpr()"
vim.opt.foldlevel = 99
vim.opt.guifont = "CaskaydiaCove Nerd Font Mono:h11"
vim.opt.clipboard = ""
vim.opt.mouse = "a"
vim.opt.ignorecase = true
vim.opt.hlsearch = true
vim.opt.incsearch = true
vim.opt.smartcase = true
vim.opt.smartindent = true
vim.opt.autoindent = true
vim.opt.colorcolumn = "99999"
vim.opt.fileencoding = "utf-8"
vim.opt.updatetime = 300
vim.opt.expandtab = false
vim.opt.shiftwidth = 4
vim.opt.tabstop = 4
vim.opt.numberwidth = 1
vim.opt.linebreak = true
vim.opt.wrap = true
vim.opt.swapfile = false
vim.opt.directory = vim.fn.stdpath("cache") .. "/swap"
vim.opt.undofile = true
vim.opt.undodir = vim.fn.stdpath("cache") .. "/undo"
vim.opt.hidden = true
vim.opt.signcolumn = "auto"
vim.opt.whichwrap = ""
vim.opt.spell = false
vim.opt.spelllang = ""
vim.opt.cursorline = true
vim.opt.termguicolors = true
vim.opt.splitbelow = true
vim.opt.splitright = true
vim.opt.title = true
vim.opt.titlestring = "%<%F%=%l/%L - lvim"
vim.opt.scrolloff = 8
vim.opt.sidescrolloff = 8
vim.opt.inccommand = "nosplit"
vim.opt.fillchars = "vert: "
vim.opt.conceallevel = 0
vim.g.vim_json_conceal = 0
vim.g.vim_json_syntax_conceal = 0
vim.g.loaded_netrw = 1
vim.g.loaded_netrwPlugin = 1
vim.opt.number = true
vim.opt.relativenumber = true
vim.opt.showcmd = true
vim.cmd([[
autocmd InsertEnter * :set norelativenumber
autocmd InsertLeave * :set relativenumber
]])
