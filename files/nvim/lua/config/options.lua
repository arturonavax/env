-- Options are automatically loaded before lazy.nvim startup
-- Default options that are always set: https://github.com/LazyVim/LazyVim/blob/main/lua/lazyvim/config/options.lua
-- Add any additional options here
--
-- Indentación base a 4 espacios
vim.opt.tabstop = 4 -- Ancho visual del carácter TAB (\t)
vim.opt.shiftwidth = 4 -- Ancho para operadores de indentación (>> y <<)
vim.opt.softtabstop = 4 -- Espacios insertados/borrados al presionar Tab/Backspace
vim.opt.expandtab = true -- Convierte tabs en espacios por defecto

-- Diferenciación visual entre tabs y espacios
vim.opt.list = true
vim.opt.listchars = {
  tab = "» ", -- Marca visualmente dónde inicia un TAB real (\t)
  trail = "·", -- Espacios sobrantes al final de línea
  nbsp = "␣", -- Espacios indivisibles
  leadmultispace = "·   ", -- Guía sutil para líneas indentadas con espacios
}

vim.opt.autoread = true

vim.g.loaded_ruby_provider = 0
vim.g.loaded_perl_provider = 0

-- Fijar el binario de Python 3
vim.g.python3_host_prog = vim.fn.exepath("python3")

-- Desactiva la línea del cursor durante renders pesados o en general
vim.opt.cursorline = false

-- Reduce la frecuencia de sondeo de eventos de actualización en idle (por defecto 4000ms)
vim.opt.updatetime = 250

vim.opt.wrap = true
vim.opt.linebreak = true -- corta por palabras completas en vez de cortar a mitad de una palabra

-- Tiempos de espera estándar para mecanografía rápida
vim.opt.timeoutlen = 300 -- Tiempo para resolver mapeos compuestos (Leader, jk, etc.)
vim.opt.ttimeoutlen = 10 -- Tiempo de espera para secuencias de escape de terminal
vim.opt.updatetime = 200 -- Escribe en disco y dispara CursorHold más rápido sin trabar la UI

-- Lista de exclusión para autocompletado nativo, wildmenu y comandos de Neovim
local exclusions = require("config.exclusions")
vim.opt.wildignore:append(exclusions.wildignore)
