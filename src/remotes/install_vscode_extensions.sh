#!/bin/bash
# Run: curl -fsSL "https://env.arturonavax.dev/install_vscode_extensions.sh" | bash
if [[ "$(command -v code)" == "" ]]; then
	echo "Visual Studio Code (code) is not installed on this system."

	exit 1
fi

code --install-extension enkia.tokyo-night                   # Tokyo Night
code --install-extension vscode-icons-team.vscode-icons      # vscode-icons
code --install-extension usernamehw.errorlens                # Error Lens
code --install-extension ryuta46.multi-command               # multi-command
code --install-extension shardulm94.trailing-spaces          # Trailing Spaces
code --install-extension vscodevim.vim                       # Vim
code --install-extension golang.go                           # Go
code --install-extension quillaja.goasm                      # Go Asm
code --install-extension dsznajder.es7-react-js-snippets     # ES7+ React/Redux/React-Native snippets
code --install-extension xabikos.JavaScriptSnippets          # JavaScript (ES6) code snippets
code --install-extension ms-vscode.vscode-typescript-next    # JavaScript and TypeScript Nightly
code --install-extension mgmcdermott.vscode-language-babel   # Babel JavaScrip
code --install-extension Pandy.bun                           # Bun
code --install-extension wix.vscode-import-cost              # Import Cost
code --install-extension Zignd.html-css-class-completion     # IntelliSense for CSS class names in HTML
code --install-extension syler.sass-indented                 # Sass
code --install-extension csstools.postcss                    # PostCSS Language Support
code --install-extension esbenp.prettier-vscode              # Prettier - Code formatter
code --install-extension rome.rome                           # Rome - Fast code formatter
code --install-extension dbaeumer.vscode-eslint              # ESLint
code --install-extension formulahendry.auto-rename-tag       # Auto Rename Tag
code --install-extension formulahendry.auto-close-tag        # Auto Close Tag
code --install-extension christian-kohler.path-intellisense  # Path Intellisense
code --install-extension ms-vscode.makefile-tools            # Makefile Tools
code --install-extension timonwong.shellcheck                # ShellCheck
code --install-extension foxundermoon.shell-format           # shell-format
code --install-extension zxh404.vscode-proto3                # vscode-proto3
code --install-extension kisstkondoros.vscode-gutter-preview # Image preview
code --install-extension shd101wyy.markdown-preview-enhanced # Markdown Preview Enhanced
code --install-extension humao.rest-client                   # Rest Client
code --install-extension rangav.vscode-thunder-client        # Thunder Client
code --install-extension bufbuild.vscode-buf                 # Buf
code --install-extension pranaygp.vscode-css-peek            # CSS Peek
code --install-extension mariusalchimavicius.json-to-ts      # JSON to TS
