lvim.builtin.telescope.defaults.layout_strategy = "horizontal"
lvim.builtin.telescope.defaults.layout_config = {
	width = 0.7,
	height = 0.7,
	prompt_position = "top",
}

lvim.builtin.telescope.defaults.prompt_prefix = "🔍 "
lvim.builtin.telescope.defaults.preview = {
	filesize_limit = 1,
	timeout = 200,
}

lvim.builtin.telescope.defaults.vimgrep_arguments = {
	"rg",
	"--color=never",
	"--no-heading",
	"--with-filename",
	"--line-number",
	"--column",
	"--max-columns=300",
	"--smart-case",
	"--hidden",
	"-g",
	"!package-lock.json",
	"-g",
	"!go.sum",
	"-g",
	"!node_modules",
	"-g",
	"!.git",
}

lvim.builtin.telescope.defaults.mappings = {
	n = {
		["q"] = require("telescope.actions").close,
		["<Esc>"] = require("telescope.actions").close,
		["t"] = require("telescope.actions").select_tab,
		["v"] = require("telescope.actions").select_vertical,
		["s"] = require("telescope.actions").select_horizontal,
		["<C-d>"] = require("telescope.actions").delete_buffer,
	},
	i = {
		["<C-t>"] = require("telescope.actions").select_tab,
		["<C-v>"] = require("telescope.actions").select_vertical,
		["<C-s>"] = require("telescope.actions").select_horizontal,
		["<C-d>"] = require("telescope.actions").delete_buffer,
	},
}
