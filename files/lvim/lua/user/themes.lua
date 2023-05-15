function Dark_theme()
	vim.opt.background = "dark"
	lvim.colorscheme = "tokyonight-night"

	require("tokyonight").setup({
		style = "night",
		light_style = "day",
		dim_inactive = true,
		styles = {
			functions = { italic = true, bold = true },
			comments = { italic = true },
			keywords = { italic = true },
		},
	})

	vim.cmd([[
	" CursorLine color
	autocmd ColorScheme * highlight CursorLine guibg=#222330

	" SignColumn color
	autocmd ColorScheme * highlight SignColumn guibg=NONE

	" WinSeparator color
	autocmd ColorScheme * highlight WinSeparator guifg=#1a1b26
	autocmd ColorScheme * highlight WinSeparator guibg=NONE

	" dashboard
	autocmd ColorScheme * highlight Label guifg=#BF4B4B gui=bold,italic

	" treesitter-context
	autocmd ColorScheme * highlight TreesitterContext guibg=#292E42 gui=bold,italic
	autocmd ColorScheme * highlight TreesitterContextLineNumber guibg=#292E42 guifg=white

	" indent_blankline
	autocmd ColorScheme * highlight IndentBlanklineContextStart guisp=#7094E1 gui=underline
	autocmd ColorScheme * highlight IndentBlanklineContextChar guifg=#7094E1 gui=nocombine

	" illuminate
	autocmd ColorScheme * highlight IlluminatedWordText guibg=#222330 guisp=#7094E1 gui=underline

	" nvim-tree
	autocmd ColorScheme * highlight NvimTreeNormal guibg=NONE
	autocmd ColorScheme * highlight NvimTreeFolderIcon guifg=#ECBE7B

	"" enlarge icons with certain patched fonts
	" autocmd ColorScheme * highlight NvimTreeSpecialFile gui=bold
	" autocmd ColorScheme * highlight NvimTreeFolderIcon gui=italic
	" autocmd ColorScheme * highlight NvimTreeFileIcon gui=italic
	" autocmd ColorScheme * highlight NvimTreeOpenedFolderName gui=bold guifg=lightgray

	" bufferline
	autocmd ColorScheme * highlight BufferLineIndicatorSelected guifg=#BF4B4B
	autocmd ColorScheme * highlight BufferLineFill guibg=#16161E
	autocmd ColorScheme * highlight TabLineFill guibg=#16161E gui=bold
	]])

	-- local normal_white = "NONE"
	local main_red = "#BF4B4B"

	Custom_tokyonight_night = require("lualine.themes.tokyonight")
	Custom_tokyonight_night.normal.a.bg = main_red
	Custom_tokyonight_night.normal.b.bg = "#252636"
	Custom_tokyonight_night.insert.b.bg = "#252636"
	Custom_tokyonight_night.visual.b.bg = "#252636"
	Custom_tokyonight_night.command.b.bg = "#252636"
	Custom_tokyonight_night.replace.b.bg = "#252636"

	lvim.builtin.lualine.options.theme = Custom_tokyonight_night

	vim.g.better_whitespace_ctermcolor = "darkred"
	vim.g.better_whitespace_guicolor = "#3C4461"
end

function Light_theme()
	vim.opt.background = "light"
	lvim.colorscheme = "tokyonight-day"

	require("tokyonight").setup({
		style = "night",
		light_style = "day",
		dim_inactive = true,
		day_brightness = 0.2,
		styles = {
			functions = { italic = true, bold = true },
			comments = { italic = true },
			keywords = { italic = true },
		},
	})

	vim.cmd([[
	" WinSeparator color
	autocmd ColorScheme * highlight WinSeparator guibg=#E1E2E7 guifg=#E1E2E7

	" dashboard
	autocmd ColorScheme * highlight Label guifg=#BF4B4B gui=bold,italic

	" treesitter-context
	autocmd ColorScheme * highlight TreesitterContext guibg=#F2F2F5 gui=bold,italic
	autocmd ColorScheme * highlight TreesitterContextLineNumber guibg=#F2F2F5 guifg=darkblue

	" indent_blankline
	autocmd ColorScheme * highlight IndentBlanklineContextStart guisp=#7094E1 gui=underline
	autocmd ColorScheme * highlight IndentBlanklineContextChar guifg=#7094E1 gui=nocombine

	" illuminate
	autocmd ColorScheme * highlight IlluminatedWordText guibg=#aeaeb5 guisp=#7094E1 gui=underline

	" nvim-tree
	autocmd ColorScheme * highlight NvimTreeNormal guibg=NONE
	autocmd ColorScheme * highlight NvimTreeFolderIcon guifg=#ECBE7B

	"" enlarge icons with certain patched fonts
	" autocmd ColorScheme * highlight NvimTreeSpecialFile gui=bold
	" autocmd ColorScheme * highlight NvimTreeFolderIcon gui=italic
	" autocmd ColorScheme * highlight NvimTreeFileIcon gui=italic
	" autocmd ColorScheme * highlight NvimTreeOpenedFolderName gui=bold guifg=lightgray

	" bufferline
	autocmd ColorScheme * highlight BufferLineIndicatorSelected guifg=#BF4B4B
	autocmd ColorScheme * highlight BufferLineFill guibg=#F2F2F5
	autocmd ColorScheme * highlight TabLineFill guibg=NONE gui=bold
	]])

	-- local normal_white = "NONE"
	local no_normal_white = "#F2F2F5"
	local main_red = "#BF4B4B"

	lvim.builtin.bufferline.highlights.separator = { bg = no_normal_white }
	lvim.builtin.bufferline.highlights.warning = { bg = no_normal_white }
	lvim.builtin.bufferline.highlights.close_button = { bg = no_normal_white }
	lvim.builtin.bufferline.highlights.background = { bg = no_normal_white }
	lvim.builtin.bufferline.highlights.warning_diagnostic = { bg = no_normal_white }

	Custom_tokyonight_night = require("lualine.themes.tokyonight")
	Custom_tokyonight_night.normal.a.bg = main_red
	Custom_tokyonight_night.normal.b.bg = "#CCCCCC"
	Custom_tokyonight_night.insert.b.bg = "#CCCCCC"
	Custom_tokyonight_night.visual.b.bg = "#CCCCCC"
	Custom_tokyonight_night.command.b.bg = "#CCCCCC"
	Custom_tokyonight_night.replace.b.bg = "#CCCCCC"

	lvim.builtin.lualine.options.theme = Custom_tokyonight_night

	vim.g.better_whitespace_ctermcolor = "darkred"
	vim.g.better_whitespace_guicolor = "#3C4461"
end

local home_path = os.getenv("HOME")
local file_theme_mode = io.open(home_path .. "/.theme_mode", "r")

if file_theme_mode then
	local content = file_theme_mode.read(file_theme_mode)
	file_theme_mode:close()

	if content == "light" then
		Light_theme()
	else
		Dark_theme()
	end
else
	Dark_theme()
end
