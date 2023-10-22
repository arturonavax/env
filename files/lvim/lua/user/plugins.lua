-- core plugins
lvim.builtin.which_key.active = false
lvim.builtin.breadcrumbs.active = false
lvim.builtin.lir.active = false
lvim.builtin.cmp.completion.keyword_length = 1

---- project.nvim
lvim.builtin.project.show_hidden = true
lvim.builtin.project.patterns = Root_patterns

---- null-ls
require("null-ls").setup({
	root_dir = require("null-ls.utils").root_pattern(unpack(Root_patterns)),
	-- temp_dir = vim.fn.stdpath("cache"), -- May cause problems with some sources of null-ls
})

---- gitsigns
lvim.builtin.gitsigns.opts.signs.delete.text = "▎"

---- gitblame
vim.g.gitblame_enabled = 0

---- indent_blankline
lvim.builtin.indentlines.options.show_current_context = true
lvim.builtin.indentlines.options.use_treesitter = true
lvim.builtin.indentlines.options.show_trailing_blankline_indent = false
lvim.builtin.indentlines.options.show_first_indent_level = false

-- extra plugins
lvim.plugins = {
	---- ~/.lunarvim.lua: table.insert(lvim.plugins, { "wakatime/vim-wakatime" })
	{ "tpope/vim-repeat" },
	{
		"zbirenbaum/copilot-cmp",
		event = "InsertEnter",
		dependencies = { "zbirenbaum/copilot.lua" },
		config = function()
			vim.defer_fn(function()
				require("copilot").setup() -- https://github.com/zbirenbaum/copilot.lua/blob/master/README.md#setup-and-configuration
				require("copilot_cmp").setup() -- https://github.com/zbirenbaum/copilot-cmp/blob/master/README.md#configuration
			end, 100)
		end,
	},
	{
		"kevinhwang91/nvim-ufo",
		dependencies = "kevinhwang91/promise-async",
		config = function()
			local handler = function(virtText, lnum, endLnum, width, truncate)
				local newVirtText = {}
				local suffix = (" ⋯   %d "):format(endLnum - lnum)
				local sufWidth = vim.fn.strdisplaywidth(suffix)
				local targetWidth = width - sufWidth
				local curWidth = 0
				for _, chunk in ipairs(virtText) do
					local chunkText = chunk[1]
					local chunkWidth = vim.fn.strdisplaywidth(chunkText)
					if targetWidth > curWidth + chunkWidth then
						table.insert(newVirtText, chunk)
					else
						chunkText = truncate(chunkText, targetWidth - curWidth)
						local hlGroup = chunk[2]
						table.insert(newVirtText, { chunkText, hlGroup })
						chunkWidth = vim.fn.strdisplaywidth(chunkText)
						-- str width returned from truncate() may less than 2nd argument, need padding
						if curWidth + chunkWidth < targetWidth then
							suffix = suffix .. (" "):rep(targetWidth - curWidth - chunkWidth)
						end
						break
					end
					curWidth = curWidth + chunkWidth
				end
				table.insert(newVirtText, { suffix, "MoreMsg" })
				return newVirtText
			end

			require("ufo").setup({
				enable_get_fold_virt_text = true,
				fold_virt_text_handler = handler,
				provider_selector = function(_, _, _)
					return { "treesitter", "indent" }
				end,
			})

			-- vim.o.fillchars = [[eob: ,fold: ,foldopen:▼,foldsep: ,foldclose:⏵]]
			vim.o.foldcolumn = "0" -- '0' is not bad
			vim.o.foldlevel = 99 -- Using ufo provider need a large value, feel free to decrease the value
			vim.o.foldlevelstart = 99
			vim.o.foldenable = true

			vim.keymap.set("n", "zR", require("ufo").openAllFolds)
			vim.keymap.set("n", "zM", require("ufo").closeAllFolds)
			vim.keymap.set("n", "zr", require("ufo").openFoldsExceptKinds)
			vim.keymap.set("n", "zm", require("ufo").closeFoldsWith) -- closeAllFolds == closeFoldsWith(0)
		end,
	},
	{
		"ntpeters/vim-better-whitespace",
		config = function()
			vim.g.better_whitespace_enabled = 1
			vim.g.better_whitespace_filetypes_blacklist = {
				"diff",
				"help",
				"fugitive",
				"NvimTree",
				"lspinfo",
				"lsp-installer",
				"dashboard",
				"alpha",
			}
		end,
	},
	{ "tpope/vim-surround" },
	{ "p00f/nvim-ts-rainbow" },
	{
		"ggandor/leap.nvim",
		name = "leap",
		config = function()
			require("leap").add_default_mappings()
		end,
	},
	{
		"kazhala/close-buffers.nvim",
		config = function()
			require("close_buffers").setup({
				filetype_ignore = {}, -- Filetype to ignore when running deletions
				file_glob_ignore = {}, -- File name glob pattern to ignore when running deletions (e.g. '*.md')
				file_regex_ignore = {}, -- File name regex pattern to ignore when running deletions (e.g. '.*[.]md')
				preserve_window_layout = { "this", "nameless" }, -- Types of deletion that should preserve the window layout
				next_buffer_cmd = nil, -- Custom function to retrieve the next buffer when preserving window layout
			})
		end,
	},
	{
		"ThePrimeagen/refactoring.nvim",
		dependencies = {
			{ "nvim-lua/plenary.nvim" },
			{ "nvim-treesitter/nvim-treesitter" },
		},
		config = true,
	},
	{
		"folke/lsp-colors.nvim",
		event = "BufRead",
	},
	{
		"felipec/vim-sanegx",
		event = "BufRead",
	},
	{
		"ray-x/lsp_signature.nvim",
		event = "BufRead",
		config = function()
			require("lsp_signature").setup({
				always_trigger = true,
				hint_prefix = "🦊 ",
				doc_lines = 0,
			})
		end,
	},
	{
		"andymass/vim-matchup",
		event = "CursorMoved",
		config = function()
			vim.g.matchup_matchparen_offscreen = { method = "popup" }
		end,
	},
	{
		"norcalli/nvim-colorizer.lua",
		config = function()
			require("colorizer").setup(
				{ "css", "scss", "html", "javascript", "typescript", "javascriptreact", "typescriptreact" },
				{
					RGB = true, -- #RGB hex codes
					RRGGBB = true, -- #RRGGBB hex codes
					RRGGBBAA = true, -- #RRGGBBAA hex codes
					rgb_fn = true, -- CSS rgb() and rgba() functions
					hsl_fn = true, -- CSS hsl() and hsla() functions
					css = true, -- Enable all CSS features: rgb_fn, hsl_fn, names, RGB, RRGGBB
					css_fn = true, -- Enable all CSS *functions*: rgb_fn, hsl_fn
					names = true, -- "Name" codes like Blue
				}
			)
		end,
	},
	{
		"kevinhwang91/nvim-bqf",
		event = { "BufRead", "BufNew" },
		config = function()
			require("bqf").setup({
				auto_enable = true,
				preview = {
					win_height = 12,
					win_vheight = 12,
					delay_syntax = 80,
					-- border_chars = { "┃", "┃", "━", "━", "┏", "┓", "┗", "┛", "█" },
					border_chars = { "│", "│", "─", "─", "╭", "╮", "╰", "╯", "█" },
				},
				func_map = {
					vsplit = "v",
					split = "s",
					tab = "t",
					ptogglemode = "z,",
					stoggleup = "",
				},
				filter = {
					fzf = {
						action_for = { ["ctrl-s"] = "split" },
						extra_opts = { "--bind", "ctrl-o:toggle-all", "--prompt", "> " },
					},
				},
			})
		end,
	},
	{
		"folke/persistence.nvim",
		event = "BufReadPre", -- this will only start session saving when an actual file was opened
		lazy = true,
		config = function()
			require("persistence").setup({
				dir = vim.fn.expand(vim.fn.stdpath("config") .. "/session/"),
				options = { "buffers", "curdir", "tabpages", "winsize" },
			})
		end,
	},
	{
		"folke/todo-comments.nvim",
		event = "BufRead",
		config = function()
			require("todo-comments").setup({
				highlight = {
					keyword = "bg",
				},
			})
		end,
	},

	-- treesitter
	{
		"romgrk/nvim-treesitter-context",
		config = function()
			require("treesitter-context").setup({
				enable = true,
				max_lines = 0,
				throttle = true,
				line_numbers = true,
				multiline_threshold = 15,
				trim_scope = "outer",
				mode = "cursor",
				patterns = {
					default = {
						"class",
						"function",
						"method",
						"for",
						"while",
						"if",
						"switch",
						"case",
					},
				},
			})
		end,
	},
	{
		"windwp/nvim-ts-autotag",
		config = true,
	},

	-- http
	{
		"rest-nvim/rest.nvim",
		dependencies = { "nvim-lua/plenary.nvim" },
		config = function()
			require("rest-nvim").setup({
				-- Open request results in a horizontal split
				result_split_horizontal = false,
				-- Keep the http file buffer above|left when split horizontal|vertical
				result_split_in_place = false,
				-- Skip SSL verification, useful for unknown certificates
				skip_ssl_verification = false,
				-- Encode URL before making request
				encode_url = true,
				-- Highlight request on run
				highlight = {
					enabled = true,
					timeout = 150,
				},
				result = {
					-- toggle showing URL, HTTP info, headers at top the of result window
					show_url = true,
					show_http_info = true,
					show_headers = true,
					-- executables or functions for formatting response body [optional]
					-- set them to false if you want to disable them
					formatters = {
						json = "jq",
						html = function(body)
							return vim.fn.system({ "tidy", "-i", "-q", "-" }, body)
						end,
					},
				},
				-- Jump to request line on run
				jump_to_request = false,
				env_file = ".env",
				custom_dynamic_variables = {},
				yank_dry_run = true,
			})
		end,
	},

	-- go
	{
		"leoluz/nvim-dap-go",
		config = true,
	},
	{
		"ray-x/guihua.lua",
		build = "cd lua/fzy && make",
		config = function()
			vim.cmd("autocmd FileType guihua nmap <buffer> q <ESC>")
		end,
	},
	{
		"ray-x/go.nvim",
		ft = { "go", "gomod" },
		event = { "CmdlineEnter" },
		build = ':lua require("go.install").update_all_sync()',
		dependencies = {
			"ray-x/guihua.lua",
			"neovim/nvim-lspconfig",
			"nvim-treesitter/nvim-treesitter",
			"theHamsta/nvim-dap-virtual-text",
			"rcarriga/nvim-dap-ui",
			"leoluz/nvim-dap-go",
			"mfussenegger/nvim-dap",
		},
		config = function()
			require("go").setup({
				max_line_len = 120,
				tag_transform = "camelcase", -- false, "snakecase", "camelcase", "lispcase", "pascalcase", "titlecase", "keep"
				tag_options = "",
				comment_placeholder = "",
				sign_priority = 12,
			})

			vim.cmd([[
			autocmd FileType go nmap <buffer> <leader>gj :GoBuild<CR>
			autocmd FileType go nmap <buffer> <leader>gl :GoLint<CR>
			autocmd FileType go nmap <buffer> <leader>gat :GoAddTag<CR>
			autocmd FileType go nmap <buffer> <leader>grt :GoRmTag<CR>

			autocmd FileType go nmap <buffer> <leader>T :GoTest<CR>
			autocmd FileType go nmap <buffer> <leader>gtt :GoAddAllTest<CR>
			autocmd FileType go nmap <buffer> <leader>t :GoAltV<CR>
			autocmd FileType go nmap <buffer> <leader>gac :GoCmt<CR>
			autocmd FileType go nmap <buffer> <leader>gtc :GoCoverage<CR>
			autocmd FileType go nmap <buffer> <leader>c :GoCoverage -t<CR>
			autocmd FileType go nmap <buffer> <leader>gfs :GoFillStruct<CR>
			autocmd FileType go nmap <buffer> <leader>gfw :GoFillSwitch<CR>
			]])

			---- Run gofmt + goimport on save
			local format_sync_grp = vim.api.nvim_create_augroup("GoImport", {})
			vim.api.nvim_create_autocmd("BufWritePre", {
				pattern = "*.go",
				callback = function()
					require("go.format").goimport()
				end,
				group = format_sync_grp,
			})
		end,
	},

	-- markdown
	{ "mzlogin/vim-markdown-toc", ft = "markdown" },
	{
		"iamcco/markdown-preview.nvim",
		build = "cd app && npm install",
		ft = "markdown",
		config = function()
			vim.g.mkdp_auto_start = 1
		end,
	},

	-- git
	{
		"tpope/vim-fugitive",
		cmd = {
			"G",
			"Git",
			"Gdiffsplit",
			"Gread",
			"Gwrite",
			"Ggrep",
			"GMove",
			"GDelete",
			"GBrowse",
			"GRemove",
			"GRename",
			"Glgrep",
			"Gedit",
		},
		ft = { "fugitive" },
	},
	{
		"f-person/git-blame.nvim",
		event = "BufRead",
		config = function()
			vim.cmd("highlight default link gitblame SpecialComment")
			require("gitblame").setup({ enabled = false })
		end,
	},
	{
		"sindrets/diffview.nvim",
		event = "BufRead",
	},

	-- rust
	{
		"simrat39/rust-tools.nvim",
		-- ft = { "rust", "rs" },
		config = function()
			require("rust-tools").setup({
				tools = {
					executor = require("rust-tools/executors").termopen, -- can be quickfix or termopen
					reload_workspace_from_cargo_toml = true,
					runnables = {
						use_telescope = true,
					},
					inlay_hints = {
						auto = true,
						only_current_line = false,
						show_parameter_hints = true,
						parameter_hints_prefix = "<-",
						other_hints_prefix = "=>",
						max_len_align = false,
						max_len_align_padding = 1,
						right_align = false,
						right_align_padding = 7,
						highlight = "Comment",
					},
					hover_actions = {
						border = {
							{ "╭", "FloatBorder" },
							{ "─", "FloatBorder" },
							{ "╮", "FloatBorder" },
							{ "│", "FloatBorder" },
							{ "╯", "FloatBorder" },
							{ "─", "FloatBorder" },
							{ "╰", "FloatBorder" },
							{ "│", "FloatBorder" },
						},
						auto_focus = true,
					},
				},
				server = {
					on_attach = require("lvim.lsp").common_on_attach,
					on_init = require("lvim.lsp").common_on_init,

					settings = {
						["rust-analyzer"] = {
							checkOnSave = {
								command = "clippy",
							},
						},
					},
				},
			})
		end,
	},
}

-- olds plugins and configs
-- {
-- 	"fatih/vim-go",
-- 	config = function()
-- 		vim.g.go_fmt_command = ""
-- 		vim.g.go_fmt_autosave = 0
-- 		vim.g.go_mod_fmt_autosave = 0
-- 		vim.g.go_asmfmt_autosave = 0
-- 		vim.g.go_metalinter_autosave = 0

-- 		vim.g.go_addtags_transform = "camelcase"
-- 		vim.g.go_auto_type_info = 0
-- 		vim.g.go_jump_to_error = 0
-- 		vim.g.go_auto_sameids = 0
-- 		vim.g.go_highlight_types = 1
-- 		vim.g.go_highlight_fields = 1
-- 		vim.g.go_highlight_functions = 1
-- 		vim.g.go_highlight_function_calls = 1
-- 		vim.g.go_highlight_operators = 1
-- 		vim.g.go_highlight_extra_types = 1
-- 		vim.g.go_highlight_build_constraints = 1
-- 		vim.g.go_highlight_generate_tags = 1
-- 		vim.g.go_term_enabled = 1

-- 		-- vim.cmd([[
-- 		-- autocmd FileType go nmap <buffer> <leader>gj :GoBuild<CR>
-- 		-- autocmd FileType go nmap <buffer> <leader>gk :GoVet<CR>
-- 		-- autocmd FileType go nmap <buffer> <leader>gl :GoLint<CR>
-- 		-- autocmd FileType go nmap <buffer> <leader>gat :GoAddTags<CR>
-- 		-- autocmd FileType go nmap <buffer> <leader>grt :GoRemoveTags<CR>
-- 		-- autocmd FileType go nmap <buffer> <leader>gcp :GoChannelPeers<CR>
-- 		-- ]])
-- 	end,
-- },
