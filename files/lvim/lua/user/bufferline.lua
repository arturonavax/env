lvim.builtin.bufferline.options.buffer_close_icon = ""
lvim.builtin.bufferline.options.close_icon = ""
lvim.builtin.bufferline.options.separator_style = { " ", " " }

if lvim.builtin.bufferline.options.offsets[2].filetype == "NvimTree" then
	lvim.builtin.bufferline.options.offsets[2].text = " File Explorer"
end

lvim.builtin.bufferline.options.always_show_bufferline = true
