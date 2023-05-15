lvim.builtin.lualine.style = "lvim"

local components = require("lvim.core.lualine.components")

local component_branch = components.branch
component_branch.icon = ""
component_branch.padding = 1

local component_diff = components.diff
component_diff.padding = 1
component_diff.symbols.added = " "
component_diff.diff_color.added.fg = "#6a8a3f"
component_diff.diff_color.modified.fg = "#b88235"

lvim.builtin.lualine.sections.lualine_b = {
	component_branch,
	component_diff,
}

local component_filename = components.filename
component_filename.path = 1
component_filename.symbols = {
	modified = " [●]",
	readonly = " []",
	unnamed = " [No Name]",
}

lvim.builtin.lualine.sections.lualine_c = {
	component_filename,
}

lvim.builtin.lualine.sections.lualine_x = {
	components.diagnostics,
}
lvim.builtin.lualine.sections.lualine_y = {
	components.filetype,
}

local component_location = components.location
component_location.padding = 1

lvim.builtin.lualine.sections.lualine_z = {
	component_location,
	components.scrollbar,
}
