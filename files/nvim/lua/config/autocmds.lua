-- Autocmds are automatically loaded on the VeryLazy event
-- Default autocmds that are always set: https://github.com/LazyVim/LazyVim/blob/main/lua/lazyvim/config/autocmds.lua
--
-- Add any additional autocmds here
-- with `vim.api.nvim_create_autocmd`
--
-- Or remove existing autocmds by their group name (which is prefixed with `lazyvim_` for the defaults)
-- e.g. vim.api.nvim_del_augroup_by_name("lazyvim_wrap_spell")

vim.api.nvim_create_autocmd({ "FocusGained", "BufEnter", "CursorHold" }, {
  pattern = "*",
  command = "if mode() != 'c' | checktime | endif",
})

-- Desactiva diagnósticos y formateo automático al inspeccionar archivos de vendors o carpetas excluidas
local exclusions = require("config.exclusions")
vim.api.nvim_create_autocmd({ "BufReadPost", "BufNewFile", "LspAttach" }, {
  group = vim.api.nvim_create_augroup("user_exclusions_buffer_settings", { clear = true }),
  callback = function(args)
    local name = vim.api.nvim_buf_get_name(args.buf)
    if exclusions.is_path_excluded(name) then
      vim.b[args.buf].autoformat = false
      pcall(vim.diagnostic.enable, false, { bufnr = args.buf })
    end
  end,
})
