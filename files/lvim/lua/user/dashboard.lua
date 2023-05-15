lvim.builtin.alpha.dashboard.section.header.val = {
	"                                            /\\   /\\   Hi! 🦊, " .. vim.fn.expand("$USER"),
	"                                           //\\\\_//\\\\     ____",
	"                                           \\_     _/    /   /",
	"                                            / * * \\    /^^^]",
	"                                            \\_\\O/_/    [   ]",
	"                                             /   \\_    [   /",
	"                                             \\     \\_  /  /",
	"                                              [ [ /  \\/ _/",
	"                                             _[ [ \\  /_/",
	"██╗     ██╗   ██╗███╗   ██╗ █████╗ ██████╗ ██╗   ██╗██╗███╗   ███╗",
	"██║     ██║   ██║████╗  ██║██╔══██╗██╔══██╗██║   ██║██║████╗ ████║",
	"██║     ██║   ██║██╔██╗ ██║███████║██████╔╝██║   ██║██║██╔████╔██║",
	"██║     ██║   ██║██║╚██╗██║██╔══██║██╔══██╗╚██╗ ██╔╝██║██║╚██╔╝██║",
	"███████╗╚██████╔╝██║ ╚████║██║  ██║██║  ██║ ╚████╔╝ ██║██║ ╚═╝ ██║",
	"╚══════╝ ╚═════╝ ╚═╝  ╚═══╝╚═╝  ╚═╝╚═╝  ╚═╝  ╚═══╝  ╚═╝╚═╝     ╚═╝",
}

local function create_dashboard_button(name, shortcut, command)
	return {
		val = name,
		type = "button",
		on_press = function()
			local key = vim.api.nvim_replace_termcodes(shortcut, true, false, true)

			vim.api.nvim_feedkeys(key, "normal", false)
		end,
		opts = {
			shortcut = shortcut,
			keymap = {
				"n",
				shortcut,
				command,
				{
					noremap = true,
					nowait = true,
					silent = true,
				},
			},
			cursor = 5,
			width = 50,
			position = "center",
			align_shortcut = "right",
		},
	}
end

lvim.builtin.alpha.dashboard.section.buttons.val = {
	create_dashboard_button("󰳶  Last Session in Current Folder", ";", "<CMD>lua require('persistence').load()<CR>"),
	create_dashboard_button("󰥓  Last Session", "'", "<CMD>lua require('persistence').load({ last = true })<CR>"),
	create_dashboard_button("  Projects", "p", "<CMD>Telescope projects<CR>"),
	create_dashboard_button("󰇗  Open Main File of this folder", "o", "<CMD>lua OpenMainFile()<CR>"),
	create_dashboard_button("  New File", "i", "<CMD>ene!<CR>i"),
	create_dashboard_button("  File Explorer", "e", "<CMD>NvimTreeOpen<CR>"),
	create_dashboard_button("  Git Diff", "d", "<CMD>DiffviewOpen<CR>"),
	create_dashboard_button("  Find File", "f", "<CMD>Telescope find_files<CR>"),
	create_dashboard_button("  Find Word", "w", "<CMD>Telescope live_grep<CR>"),
	create_dashboard_button("  Recently Used Files", "u", "<CMD>Telescope oldfiles<CR>"),
	create_dashboard_button("  Open X Folder", "x", "<CMD>silent exec '!open .'<CR>"),
	create_dashboard_button("❤️  Checkhealth", "s", "<CMD>checkhealth<CR>"),
	create_dashboard_button("  Configuration", "c", "<CMD>edit" .. vim.fn.stdpath("config") .. "/config.lua<CR>"),
	create_dashboard_button("❌ Exit", "q", "<CMD>q<CR>"),
}

lvim.builtin.alpha.dashboard.section.footer.val = { "https://github.com/arturonavax/env" }
