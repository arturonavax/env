function OpenBuffersInVSCode()
	local buffers = vim.api.nvim_list_bufs()
	local fileNames = {}

	for _, buffer in ipairs(buffers) do
		if vim.api.nvim_buf_is_valid(buffer) and vim.bo[buffer].buflisted then
			local fileName = vim.api.nvim_buf_get_name(buffer)

			if vim.api.nvim_get_current_buf() == buffer then
				local location = vim.api.nvim_win_get_cursor(0)
				fileName = fileName .. ":" .. location[1] .. ":" .. location[2] + 1
				table.insert(fileNames, 1, fileName)
			else
				table.insert(fileNames, fileName)
			end
		end
	end

	local cwd = vim.fn.getcwd()
	vim.cmd("!code -g " .. cwd .. " " .. table.concat(fileNames, " "))
end

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

command! Code silent! execute "lua OpenBuffersInVSCode()"
]])
