-- Commands:
-- * Update plugins:
--  :LvimSyncCorePlugins
--
-- * Preinstall LSPs:
--  :LspAuto
--
-- * Install Go Binaries:
--  :GoInstallBinaries
--
-- * Update Go Binaries:
--  :GoUpdateBinaries

require("user.functions")
require("user.config")
require("user.plugins")
require("user.icons")
require("user.treesitter")
require("user.file_explorer")
require("user.dashboard")
require("user.bufferline")
require("user.statusline")
require("user.fuzzy_finder")
require("user.keybindings")
require("user.lsp")
require("user.code_actions")
require("user.linters")
require("user.formatters")
require("user.themes")
require("user.commands")

---- load custom configuration
local customConfig = vim.fn.glob("~/.lunarvim.lua")

if vim.fn.empty(customConfig) == 0 then
	loadfile(customConfig)()
end
