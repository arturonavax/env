vim.cmd([[
command! LspAuto silent! execute "MasonInstall gopls bash-language-server cmake-language-server dockerfile-language-server emmet-ls html-lsp css-lsp cssmodules-language-server typescript-language-server json-lsp yaml-language-server lua-language-server pyright buf-language-server"

command! Bo silent! execute "lua require('close_buffers').delete({type = 'hidden'})"

command! Cpath execute "lua Cpath()"

command! SearchWhitespace silent! /\s\+$

command! LinterRestart execute "lua require('null-ls.client')._reset()" | edit

command! RestNvim execute "lua require('rest-nvim').run()"

command! Persistence execute "lua require('persistence').load()"
command! PersistenceLast execute "lua require('persistence').load({ last = true })"
command! PersistenceStop execute "lua require('persistence').stop()"

command! DarkTheme execute "lua Dark_theme(); Dark_theme() "
command! LightTheme execute "lua Light_theme(); Light_theme()"
]])
