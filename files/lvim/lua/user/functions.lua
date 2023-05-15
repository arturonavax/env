function Cpath()
	local folder_path = vim.fn.expand("%:p:h")

	vim.fn.setreg("+", folder_path)

	vim.notify("Cpath: " .. folder_path)
end

function OpenMainFile()
	local list_main_files = {
		"^main%..+$",
		"^Main%..+$",
		"^program%..+$",
		"^Program%..+$",
		"^" .. string.gsub(vim.fn.fnamemodify(vim.fn.getcwd(), ":t"), "-", "%%-") .. "%..+$",
		"^index%..+$",
		"^serve%..+$",
		"^server%..+$",
		"^client%..+$",
		"^Makefile$",
		"^justfile$",
		"^compose%.yaml$",
		"^compose%.yml$",
		"^Dockerfile$",
	}

	local handle = io.popen("command ls -1 -p1 | grep -v /")
	if handle == nil then
		return
	end

	local files = handle:read("*a")
	handle:close()

	for file in files:gmatch("%S+") do
		for _, main_file in ipairs(list_main_files) do
			if file:match(main_file) and not file:match("index.html") then
				vim.cmd("NvimTreeOpen")
				vim.cmd("edit " .. file)

				return
			end
		end
	end

	print("No main file found")

	vim.cmd("ene!")
	vim.cmd("NvimTreeOpen")
end

function InstallLSP(server_name, callback)
	-- name checker
	local name_checker = {
		bashls = "bash-language-server",
	}

	local mason_registry = require("mason-registry")

	server_name = name_checker[server_name] or server_name

	local pkg = mason_registry.get_package(server_name)

	if pkg:is_installed() then
		callback()

		return
	end

	pkg:install():once("closed", function()
		if pkg:is_installed() then
			vim.schedule(function()
				vim.notify_once(string.format("Installation complete for [%s]", server_name), vim.log.levels.INFO)

				callback()
			end)
		end
	end)
end

function Buf_try_add(server_name, bufnr)
	bufnr = bufnr or vim.api.nvim_get_current_buf()

	require("lspconfig")[server_name].manager.try_add_wrapper(bufnr)
end

function SetCustomLSPSetup(server_name, fts, callback)
	fts = fts or {}

	vim.api.nvim_create_autocmd("FileType", {
		pattern = fts,
		callback = function()
			InstallLSP(server_name, function()
				if callback then
					callback(server_name, fts)
				else
					require("lvim.lsp.manager").setup(server_name, { filetypes = fts })

					Buf_try_add(server_name)
				end
			end)
		end,
	})
end
