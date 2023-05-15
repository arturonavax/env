lvim.lsp.automatic_configuration.skipped_filetypes = { "markdown", "rst", "plaintext", "toml" }
lvim.lsp.nlsp_settings.setup.config_home = vim.fn.stdpath("config") .. "/lsp-settings/"

-- Manual LSP configurations where a non-automatic configuration is required

---- bash
local bash_server_name = "bash-language-server"
local bash_server_name2 = "bashls"

vim.list_extend(lvim.lsp.automatic_configuration.skipped_servers, { bash_server_name })
vim.list_extend(lvim.lsp.installer.setup.automatic_installation.exclude, { bash_server_name })
vim.list_extend(lvim.lsp.automatic_configuration.skipped_servers, { bash_server_name2 })
vim.list_extend(lvim.lsp.installer.setup.automatic_installation.exclude, { bash_server_name2 })

SetCustomLSPSetup("bashls", { "sh", "bash", "zsh" }, function(server_name, fts) -- Adding "zsh" and "bash" in filetypes
	require("lvim.lsp.manager").setup(server_name, { filetypes = fts })

	Buf_try_add(server_name)
end)

---- c and cpp
local c_server_name = "clangd"
local c_filetypes = require("lvim.lsp.utils").get_supported_filetypes(c_server_name)

vim.list_extend(lvim.lsp.automatic_configuration.skipped_servers, { c_server_name })
vim.list_extend(lvim.lsp.installer.setup.automatic_installation.exclude, { c_server_name })

SetCustomLSPSetup(c_server_name, c_filetypes, function(server_name)
	local capabilities = require("lvim.lsp").common_capabilities()
	capabilities.offsetEncoding = { "utf-16" } -- Fixing offsetEncoding problem

	require("lvim.lsp.manager").setup(server_name, { capabilities = capabilities })

	Buf_try_add(server_name)
end)

---- rust
local rust_server_name = "rust_analyzer"
local rust_server_name2 = "rust-analyzer"

local rust_filetypes = require("lvim.lsp.utils").get_supported_filetypes(rust_server_name)

vim.list_extend(lvim.lsp.automatic_configuration.skipped_servers, { rust_server_name })
vim.list_extend(lvim.lsp.installer.setup.automatic_installation.exclude, { rust_server_name })
vim.list_extend(lvim.lsp.automatic_configuration.skipped_servers, { rust_server_name2 })
vim.list_extend(lvim.lsp.installer.setup.automatic_installation.exclude, { rust_server_name2 })

SetCustomLSPSetup(rust_server_name2, rust_filetypes, function(server_name)
	require("lvim.lsp.manager").setup(server_name)

	Buf_try_add(rust_server_name)
end)
