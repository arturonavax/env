-- Keymaps are automatically loaded on the VeryLazy event
-- Default keymaps that are always set: https://github.com/LazyVim/LazyVim/blob/main/lua/lazyvim/config/keymaps.lua
-- Add any additional keymaps here

-- Salir de modo insertar al estilo convencional
vim.keymap.set("i", "jk", "<Esc>", { desc = "Exit insert mode" })
vim.keymap.set("i", "kj", "<Esc>", { desc = "Exit insert mode" })

Snacks.toggle({
  name = "Git Blame Line",
  get = function()
    local ok, gs_config = pcall(require, "gitsigns.config")
    return ok and gs_config.config.current_line_blame or false
  end,
  set = function(state)
    local ok, gs = pcall(require, "gitsigns")
    if ok then
      gs.toggle_current_line_blame(state)
    end
  end,
}):map("<leader>uB") -- Mapeado a <leader>uB (o cámbialo a <leader>ug)
