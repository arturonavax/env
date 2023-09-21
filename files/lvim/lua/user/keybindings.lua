lvim.keys.insert_mode["jk"] = "<ESC>"
lvim.keys.insert_mode["kj"] = "<ESC>"
lvim.keys.normal_mode["<C-s>"] = ":w<CR>"
lvim.keys.normal_mode["Ñ"] = ":"
lvim.keys.command_mode["<C-a>"] = "<Home>"
lvim.keys.insert_mode["<C-a>"] = "<Home>"

lvim.keys.normal_mode["<leader>l"] = ":lua vim.diagnostic.setloclist()<CR>"
lvim.keys.normal_mode["g["] = ":lua vim.diagnostic.goto_prev()<CR>"
lvim.keys.normal_mode["g]"] = ":lua vim.diagnostic.goto_next()<CR>"

-- refactoring
lvim.keys.normal_mode["<leader>rr"] = ":lua require('refactoring').select_refactor()<CR>"
lvim.keys.visual_mode["<leader>rr"] = ":lua require('refactoring').select_refactor()<CR>"

-- lsp
lvim.keys.normal_mode["gd"] = ":lua vim.lsp.buf.definition()<CR>"
lvim.keys.normal_mode["gt"] = ":lua vim.lsp.buf.type_definition()<CR>"
lvim.keys.normal_mode["gi"] = ":lua vim.lsp.buf.implementation()<CR>"
lvim.keys.normal_mode["gr"] = ":lua vim.lsp.buf.references()<CR>"
lvim.keys.normal_mode["K"] = ":lua vim.lsp.buf.hover()<CR>"
lvim.keys.normal_mode["<leader>rn"] = ":lua vim.lsp.buf.rename()<CR>"

-- lvim.keys.normal_mode["<leader>a"] = ":lua vim.lsp.buf.code_action()<CR>"
-- lvim.keys.visual_mode["<leader>a"] = ":lua vim.lsp.buf.code_action()<CR>"
-- lvim.keys.normal_mode["<leader>q"] = ":lua vim.lsp.codelens.run()<CR>"

---- with go.nvim
-- lvim.keys.normal_mode["<leader>rn"] = ":GoRename<CR>"
lvim.keys.normal_mode["<leader>a"] = ":GoCodeAction<CR>"
lvim.keys.visual_mode["<leader>a"] = ":GoCodeAction<CR>"
lvim.keys.normal_mode["<leader>s"] = ":GoCodeLenAct<CR>"

-- fuzzy finder
lvim.keys.normal_mode["<leader>f"] = ":lua require('telescope.builtin').find_files()<CR>"
lvim.keys.normal_mode["<leader>j"] = ":lua require('telescope.builtin').live_grep()<CR>"
lvim.keys.normal_mode["<leader>b"] = ":lua require('telescope.builtin').buffers()<CR>"
lvim.keys.normal_mode["<leader>d"] = ":lua require('telescope.builtin').commands()<CR>"

-- nvimtree
lvim.keys.normal_mode["<leader>e"] = ":NvimTreeFindFileToggle<CR>"
lvim.keys.normal_mode["<leader>n"] = ":NvimTreeToggle<CR>"

-- bufferline
local map = vim.api.nvim_set_keymap
local opts = { noremap = true, silent = true }

map("n", "<S-h>", ":BufferLineCyclePrev<CR>", opts)
map("n", "<S-l>", ":BufferLineCycleNext<CR>", opts)
map("n", "<C-p>", ":BufferLineTogglePin<CR>", opts)
map("n", "<C-h>", ":BufferLineMovePrev<CR>", opts)
map("n", "<C-l>", ":BufferLineMoveNext<CR>", opts)
map("n", "<C-c>", ":BufferKill<CR>", opts)

map("n", "<A-1>", ":BufferLineGoToBuffer 1<CR>", opts)
map("n", "<A-2>", ":BufferLineGoToBuffer 2<CR>", opts)
map("n", "<A-3>", ":BufferLineGoToBuffer 3<CR>", opts)
map("n", "<A-4>", ":BufferLineGoToBuffer 4<CR>", opts)
map("n", "<A-5>", ":BufferLineGoToBuffer 5<CR>", opts)
map("n", "<A-6>", ":BufferLineGoToBuffer 6<CR>", opts)
map("n", "<A-7>", ":BufferLineGoToBuffer 7<CR>", opts)
map("n", "<A-8>", ":BufferLineGoToBuffer 8<CR>", opts)
map("n", "<A-9>", ":BufferLineGoToBuffer 9<CR>", opts)
map("n", "<A-0>", ":BufferLineGoToBuffer -1<CR>", opts)
