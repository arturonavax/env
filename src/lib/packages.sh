#!/usr/bin/env bash
# ==============================================================================
# src/lib/packages.sh
# Universal Multi-Platform Package Manager Abstraction & Translation Layer.
# Translates canonical package names across Debian/Ubuntu, Fedora/RHEL, Arch, and macOS.
# Part of arturonavax/env development environment.
# ==============================================================================

# Prevent double sourcing
[[ -n "${_PACKAGES_SH_LOADED:-}" ]] && return 0
_PACKAGES_SH_LOADED=1

_LIB_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
if [[ -f "${_LIB_DIR}/platform.sh" ]]; then
	# shellcheck source=/dev/null
	source "${_LIB_DIR}/platform.sh"
fi
if [[ -f "${_LIB_DIR}/logging.sh" ]]; then
	# shellcheck source=/dev/null
	source "${_LIB_DIR}/logging.sh"
fi

# ==============================================================================
# Package Translation Dictionaries
# ==============================================================================

# Debian / Ubuntu (apt) Translation
_translate_debian() {
	local canonical="$1"
	case "$canonical" in
	build-essential) echo "build-essential" ;;
	git) echo "git" ;;
	curl) echo "curl" ;;
	wget) echo "wget" ;;
	tar) echo "tar" ;;
	unzip) echo "unzip" ;;
	gzip) echo "gzip" ;;
	zstd) echo "zstd" ;;
	make) echo "make" ;;
	gcc) echo "gcc" ;;
	clang) echo "clang" ;;
	bison) echo "bison" ;;
	binutils) echo "binutils" ;;
	fontconfig) echo "fontconfig" ;;
	pkg-config) echo "pkg-config" ;;
	ca-certificates) echo "ca-certificates" ;;
	gnupg) echo "gnupg" ;;
	patch) echo "patch" ;;
	patchelf) echo "patchelf" ;;
	libevent-dev) echo "libevent-dev" ;;
	ssl-dev) echo "libssl-dev" ;;
	zlib-dev) echo "zlib1g-dev" ;;
	bz2-dev) echo "libbz2-dev" ;;
	readline-dev) echo "libreadline-dev" ;;
	sqlite-dev) echo "libsqlite3-dev" ;;
	ffi-dev) echo "libffi-dev" ;;
	ncurses-dev) echo "libncurses5-dev libncursesw5-dev" ;;
	ncurses-term) echo "ncurses-term" ;;
	xz-dev) echo "liblzma-dev xz-utils" ;;
	xml2-dev) echo "libxml2-dev libxml2-utils" ;;
	xslt-dev) echo "libxslt1-dev" ;;
	tk-dev) echo "tk-dev tk" ;;
	xcb-dev) echo "libxcb-xinerama0 libxcb-cursor0" ;;
	gtk4-dev) echo "libgtk-4-dev" ;;
	adwaita-dev) echo "libadwaita-1-dev" ;;
	gettext) echo "gettext" ;;
	wmctrl) echo "wmctrl" ;;
	xdotool) echo "xdotool" ;;
	x11-utils) echo "x11-utils" ;;
	keyd) echo "keyd" ;;
	zsh) echo "zsh" ;;
	vim) echo "vim" ;;
	nano) echo "nano" ;;
	fzf) echo "fzf" ;;
	ripgrep) echo "ripgrep" ;;
	bat) echo "bat" ;;
	eza) echo "eza" ;;
	tree) echo "tree" ;;
	jq) echo "jq" ;;
	htop) echo "htop" ;;
	rsync) echo "rsync" ;;
	xclip) echo "xclip" ;;
	xsel) echo "xsel" ;;
	direnv) echo "direnv" ;;
	shellcheck) echo "shellcheck" ;;
	nmap) echo "nmap" ;;
	cmake) echo "cmake" ;;
	autoconf) echo "autoconf" ;;
	automake) echo "automake" ;;
	python3) echo "python3" ;;
	python3-pip) echo "python3-pip" ;;
	python3-venv) echo "python3-venv" ;;
	screen) echo "screen" ;;
	watch) echo "watch" ;;
	locales) echo "locales" ;;
	language-pack-en) echo "language-pack-en" ;;
	*) echo "$canonical" ;;
	esac
}

# Fedora / RHEL / CentOS (dnf / yum) Translation
_translate_fedora() {
	local canonical="$1"
	case "$canonical" in
	build-essential) echo "@development-tools" ;;
	git) echo "git" ;;
	curl) echo "curl" ;;
	wget) echo "wget" ;;
	tar) echo "tar" ;;
	unzip) echo "unzip" ;;
	gzip) echo "gzip" ;;
	zstd) echo "zstd" ;;
	make) echo "make" ;;
	gcc) echo "gcc gcc-c++" ;;
	clang) echo "clang" ;;
	bison) echo "bison" ;;
	binutils) echo "binutils" ;;
	fontconfig) echo "fontconfig fontconfig-devel" ;;
	pkg-config) echo "pkgconf pkgconf-pkg-config" ;;
	ca-certificates) echo "ca-certificates" ;;
	gnupg) echo "gnupg2" ;;
	patch) echo "patch" ;;
	patchelf) echo "patchelf" ;;
	libevent-dev) echo "libevent-devel" ;;
	ssl-dev) echo "openssl-devel" ;;
	zlib-dev) echo "zlib zlib-devel" ;;
	bz2-dev) echo "bzip2 bzip2-devel" ;;
	readline-dev) echo "readline readline-devel" ;;
	sqlite-dev) echo "sqlite sqlite-devel" ;;
	ffi-dev) echo "libffi libffi-devel" ;;
	ncurses-dev) echo "ncurses ncurses-devel" ;;
	ncurses-term) echo "ncurses-term" ;;
	xz-dev) echo "xz xz-devel" ;;
	xml2-dev) echo "libxml2 libxml2-devel" ;;
	xslt-dev) echo "libxslt libxslt-devel" ;;
	tk-dev) echo "tk tk-devel" ;;
	xcb-dev) echo "libxcb libxcb-devel" ;;
	gtk4-dev) echo "gtk4-devel" ;;
	adwaita-dev) echo "libadwaita-devel" ;;
	gettext) echo "gettext" ;;
	wmctrl) echo "wmctrl" ;;
	xdotool) echo "xdotool" ;;
	x11-utils) echo "xorg-x11-utils" ;;
	keyd) echo "keyd" ;;
	zsh) echo "zsh" ;;
	vim) echo "vim-enhanced" ;;
	nano) echo "nano" ;;
	fzf) echo "fzf" ;;
	ripgrep) echo "ripgrep" ;;
	bat) echo "bat" ;;
	eza) echo "eza" ;;
	tree) echo "tree" ;;
	jq) echo "jq" ;;
	htop) echo "htop" ;;
	rsync) echo "rsync" ;;
	xclip) echo "xclip" ;;
	xsel) echo "xsel" ;;
	direnv) echo "direnv" ;;
	shellcheck) echo "shellcheck" ;;
	nmap) echo "nmap" ;;
	cmake) echo "cmake" ;;
	autoconf) echo "autoconf" ;;
	automake) echo "automake" ;;
	python3) echo "python3" ;;
	python3-pip) echo "python3-pip" ;;
	python3-venv) echo "python3" ;;
	screen) echo "screen" ;;
	watch) echo "procps-ng" ;;
	locales) echo "glibc-locale-source glibc-langpack-en" ;;
	language-pack-en) echo "glibc-langpack-en" ;;
	*) echo "$canonical" ;;
	esac
}

# Arch Linux / Manjaro / EndeavourOS (pacman) Translation
_translate_arch() {
	local canonical="$1"
	case "$canonical" in
	build-essential) echo "base-devel" ;;
	git) echo "git" ;;
	curl) echo "curl" ;;
	wget) echo "wget" ;;
	tar) echo "tar" ;;
	unzip) echo "unzip" ;;
	gzip) echo "gzip" ;;
	zstd) echo "zstd" ;;
	make) echo "make" ;;
	gcc) echo "gcc" ;;
	clang) echo "clang" ;;
	bison) echo "bison" ;;
	binutils) echo "binutils" ;;
	fontconfig) echo "fontconfig" ;;
	pkg-config) echo "pkgconf" ;;
	ca-certificates) echo "ca-certificates" ;;
	gnupg) echo "gnupg" ;;
	patch) echo "patch" ;;
	patchelf) echo "patchelf" ;;
	libevent-dev) echo "libevent" ;;
	ssl-dev) echo "openssl" ;;
	zlib-dev) echo "zlib" ;;
	bz2-dev) echo "bzip2" ;;
	readline-dev) echo "readline" ;;
	sqlite-dev) echo "sqlite" ;;
	ffi-dev) echo "libffi" ;;
	ncurses-dev) echo "ncurses" ;;
	ncurses-term) echo "ncurses" ;;
	xz-dev) echo "xz" ;;
	xml2-dev) echo "libxml2" ;;
	xslt-dev) echo "libxslt" ;;
	tk-dev) echo "tk" ;;
	xcb-dev) echo "libxcb" ;;
	gtk4-dev) echo "gtk4" ;;
	adwaita-dev) echo "libadwaita" ;;
	gettext) echo "gettext" ;;
	wmctrl) echo "wmctrl" ;;
	xdotool) echo "xdotool" ;;
	x11-utils) echo "xorg-xdpyinfo" ;;
	keyd) echo "keyd" ;;
	zsh) echo "zsh" ;;
	vim) echo "vim" ;;
	nano) echo "nano" ;;
	fzf) echo "fzf" ;;
	ripgrep) echo "ripgrep" ;;
	bat) echo "bat" ;;
	eza) echo "eza" ;;
	tree) echo "tree" ;;
	jq) echo "jq" ;;
	htop) echo "htop" ;;
	rsync) echo "rsync" ;;
	xclip) echo "xclip" ;;
	xsel) echo "xsel" ;;
	direnv) echo "direnv" ;;
	shellcheck) echo "shellcheck" ;;
	nmap) echo "nmap" ;;
	cmake) echo "cmake" ;;
	autoconf) echo "autoconf" ;;
	automake) echo "automake" ;;
	python3) echo "python" ;;
	python3-pip) echo "python-pip" ;;
	python3-venv) echo "python" ;;
	screen) echo "screen" ;;
	watch) echo "procps-ng" ;;
	locales) echo "" ;;
	language-pack-en) echo "" ;;
	*) echo "$canonical" ;;
	esac
}

# macOS (Darwin / Homebrew) Translation
_translate_darwin() {
	local canonical="$1"
	case "$canonical" in
	build-essential) echo "" ;; # Handled by Command Line Tools (xcode-select --install)
	git) echo "git" ;;
	curl) echo "curl" ;;
	wget) echo "wget" ;;
	tar) echo "gnu-tar" ;;
	unzip) echo "unzip" ;;
	gzip) echo "gzip" ;;
	zstd) echo "zstd" ;;
	make) echo "make" ;;
	gcc) echo "gcc" ;;
	clang) echo "" ;; # Built into macOS CLT
	bison) echo "bison" ;;
	binutils) echo "binutils" ;;
	fontconfig) echo "fontconfig" ;;
	pkg-config) echo "pkg-config" ;;
	ca-certificates) echo "ca-certificates" ;;
	gnupg) echo "gnupg" ;;
	patch) echo "gpatch" ;;
	patchelf) echo "" ;; # Linux only
	libevent-dev) echo "libevent" ;;
	ssl-dev) echo "openssl@3" ;;
	zlib-dev) echo "zlib" ;;
	bz2-dev) echo "bzip2" ;;
	readline-dev) echo "readline" ;;
	sqlite-dev) echo "sqlite" ;;
	ffi-dev) echo "libffi" ;;
	ncurses-dev) echo "ncurses" ;;
	ncurses-term) echo "" ;; # Built into macOS
	xz-dev) echo "xz" ;;
	xml2-dev) echo "libxml2" ;;
	xslt-dev) echo "libxslt" ;;
	tk-dev) echo "tcl-tk" ;;
	xcb-dev) echo "" ;;     # X11 not standard on modern macOS
	gtk4-dev) echo "gtk4" ;;
	adwaita-dev) echo "libadwaita" ;;
	gettext) echo "gettext" ;;
	wmctrl) echo "" ;;    # Linux X11 only
	xdotool) echo "" ;;   # Linux X11 only
	x11-utils) echo "" ;; # Linux X11 only
	keyd) echo "" ;;      # Linux only (Karabiner used on macOS)
	zsh) echo "zsh" ;;
	vim) echo "vim" ;;
	nano) echo "nano" ;;
	fzf) echo "fzf" ;;
	ripgrep) echo "ripgrep" ;;
	bat) echo "bat" ;;
	eza) echo "eza" ;;
	tree) echo "tree" ;;
	jq) echo "jq" ;;
	htop) echo "htop" ;;
	rsync) echo "rsync" ;;
	xclip) echo "" ;;
	xsel) echo "" ;;
	direnv) echo "direnv" ;;
	shellcheck) echo "shellcheck" ;;
	nmap) echo "nmap" ;;
	cmake) echo "cmake" ;;
	autoconf) echo "autoconf" ;;
	automake) echo "automake" ;;
	python3) echo "python@3" ;;
	python3-pip) echo "" ;; # Handled with python formula
	python3-venv) echo "" ;;
	screen) echo "screen" ;;
	watch) echo "watch" ;;
	locales) echo "" ;;
	language-pack-en) echo "" ;;
	*) echo "$canonical" ;;
	esac
}

# Android Termux (pkg) Translation
_translate_termux() {
	local canonical="$1"
	case "$canonical" in
	build-essential) echo "build-essential" ;;
	git) echo "git" ;;
	curl) echo "curl" ;;
	wget) echo "wget" ;;
	tar) echo "tar" ;;
	unzip) echo "unzip" ;;
	gzip) echo "gzip" ;;
	zstd) echo "zstd" ;;
	make) echo "make" ;;
	gcc) echo "clang" ;;
	clang) echo "clang" ;;
	bison) echo "bison" ;;
	binutils) echo "binutils" ;;
	fontconfig) echo "fontconfig" ;;
	pkg-config) echo "pkg-config" ;;
	ca-certificates) echo "ca-certificates" ;;
	gnupg) echo "gnupg" ;;
	patch) echo "patch" ;;
	patchelf) echo "patchelf" ;;
	libevent-dev) echo "libevent" ;;
	ssl-dev) echo "openssl" ;;
	zlib-dev) echo "zlib" ;;
	bz2-dev) echo "bzip2" ;;
	readline-dev) echo "readline" ;;
	sqlite-dev) echo "sqlite" ;;
	ffi-dev) echo "libffi" ;;
	ncurses-dev) echo "ncurses" ;;
	ncurses-term) echo "ncurses-utils" ;;
	xz-dev) echo "xz-utils" ;;
	xml2-dev) echo "libxml2" ;;
	xslt-dev) echo "libxslt" ;;
	tk-dev) echo "" ;;
	xcb-dev) echo "" ;;
	gtk4-dev) echo "" ;;
	adwaita-dev) echo "" ;;
	gettext) echo "gettext" ;;
	wmctrl) echo "" ;;
	xdotool) echo "" ;;
	x11-utils) echo "" ;;
	keyd) echo "" ;;
	zsh) echo "zsh" ;;
	vim) echo "vim" ;;
	nano) echo "nano" ;;
	fzf) echo "fzf" ;;
	ripgrep) echo "ripgrep" ;;
	bat) echo "bat" ;;
	eza) echo "eza" ;;
	tree) echo "tree" ;;
	jq) echo "jq" ;;
	htop) echo "htop" ;;
	rsync) echo "rsync" ;;
	xclip) echo "" ;;
	xsel) echo "" ;;
	direnv) echo "direnv" ;;
	shellcheck) echo "shellcheck" ;;
	nmap) echo "nmap" ;;
	cmake) echo "cmake" ;;
	autoconf) echo "autoconf" ;;
	automake) echo "automake" ;;
	python3) echo "python" ;;
	python3-pip) echo "python-pip" ;;
	python3-venv) echo "python" ;;
	screen) echo "screen" ;;
	watch) echo "procps" ;;
	locales) echo "" ;;
	language-pack-en) echo "" ;;
	*) echo "$canonical" ;;
	esac
}

# openSUSE / SLES (zypper) Translation
_translate_suse() {
	local canonical="$1"
	case "$canonical" in
	build-essential) echo "devel_basis" ;;
	git) echo "git" ;;
	curl) echo "curl" ;;
	wget) echo "wget" ;;
	tar) echo "tar" ;;
	unzip) echo "unzip" ;;
	gzip) echo "gzip" ;;
	zstd) echo "zstd" ;;
	make) echo "make" ;;
	gcc) echo "gcc gcc-c++" ;;
	clang) echo "clang" ;;
	bison) echo "bison" ;;
	binutils) echo "binutils" ;;
	fontconfig) echo "fontconfig fontconfig-devel" ;;
	pkg-config) echo "pkg-config" ;;
	ca-certificates) echo "ca-certificates" ;;
	gnupg) echo "gpg2" ;;
	patch) echo "patch" ;;
	patchelf) echo "patchelf" ;;
	libevent-dev) echo "libevent-devel" ;;
	ssl-dev) echo "libopenssl-devel" ;;
	zlib-dev) echo "zlib-devel" ;;
	bz2-dev) echo "libbz2-devel" ;;
	readline-dev) echo "readline-devel" ;;
	sqlite-dev) echo "sqlite3-devel" ;;
	ffi-dev) echo "libffi-devel" ;;
	ncurses-dev) echo "ncurses-devel" ;;
	ncurses-term) echo "terminfo" ;;
	xz-dev) echo "xz-devel" ;;
	xml2-dev) echo "libxml2-devel" ;;
	xslt-dev) echo "libxslt-devel" ;;
	tk-dev) echo "tk-devel" ;;
	xcb-dev) echo "libxcb-devel" ;;
	gtk4-dev) echo "gtk4-devel" ;;
	adwaita-dev) echo "libadwaita-devel" ;;
	gettext) echo "gettext-tools" ;;
	wmctrl) echo "wmctrl" ;;
	xdotool) echo "xdotool" ;;
	x11-utils) echo "xorg-x11-server-extra" ;;
	keyd) echo "keyd" ;;
	zsh) echo "zsh" ;;
	vim) echo "vim" ;;
	nano) echo "nano" ;;
	fzf) echo "fzf" ;;
	ripgrep) echo "ripgrep" ;;
	bat) echo "bat" ;;
	eza) echo "eza" ;;
	tree) echo "tree" ;;
	jq) echo "jq" ;;
	htop) echo "htop" ;;
	rsync) echo "rsync" ;;
	xclip) echo "xclip" ;;
	xsel) echo "xsel" ;;
	direnv) echo "direnv" ;;
	shellcheck) echo "ShellCheck" ;;
	nmap) echo "nmap" ;;
	cmake) echo "cmake" ;;
	autoconf) echo "autoconf" ;;
	automake) echo "automake" ;;
	python3) echo "python3" ;;
	python3-pip) echo "python3-pip" ;;
	python3-venv) echo "python3" ;;
	screen) echo "screen" ;;
	watch) echo "procps" ;;
	locales) echo "glibc-locale" ;;
	language-pack-en) echo "" ;;
	*) echo "$canonical" ;;
	esac
}

# Translate Canonical Name to Distro-Native Package
pkg_translate() {
	local canonical="$1"
	local distro
	distro="$(detect_distro)"

	case "$distro" in
	termux)
		_translate_termux "$canonical"
		;;
	ubuntu | debian)
		_translate_debian "$canonical"
		;;
	fedora | rhel)
		_translate_fedora "$canonical"
		;;
	arch)
		_translate_arch "$canonical"
		;;
	suse)
		_translate_suse "$canonical"
		;;
	darwin)
		_translate_darwin "$canonical"
		;;
	*)
		# Fallback to debian if unknown linux
		if [[ "$(detect_os)" == "linux" ]]; then
			_translate_debian "$canonical"
		else
			echo "$canonical"
		fi
		;;
	esac
}

# ==============================================================================
# Unified Package Manager Operations
# ==============================================================================

# Check if a specific native package is installed
pkg_is_installed() {
	local pkg="$1"
	local mgr
	mgr="$(detect_pkg_mgr)"

	case "$mgr" in
	termux | apt)
		dpkg-query -W -f='${Status}' "$pkg" 2>/dev/null | grep -q "install ok installed"
		;;
	dnf | yum | zypper)
		rpm -q "$pkg" >/dev/null 2>&1
		;;
	pacman)
		pacman -Qi "$pkg" >/dev/null 2>&1
		;;
	apk)
		apk info -e "$pkg" >/dev/null 2>&1
		;;
	brew)
		brew list "$pkg" >/dev/null 2>&1 || brew list --cask "$pkg" >/dev/null 2>&1
		;;
	*)
		command -v "$pkg" >/dev/null 2>&1
		;;
	esac
}

# Update Package Repository Indexes
pkg_update_cache() {
	local mgr
	mgr="$(detect_pkg_mgr)"
	log_info "Updating package indexes using ${BOLD_WHITE}${mgr}${RESET}..."

	local sudo_cmd="sudo"
	[[ $EUID -eq 0 ]] && sudo_cmd=""

	case "$mgr" in
	termux)
		pkg update -y
		;;
	apt)
		$sudo_cmd apt update -y
		;;
	dnf)
		# dnf check-update returns exit code 100 when updates are available
		$sudo_cmd dnf check-update -y || [[ $? -eq 100 ]] || true
		;;
	pacman)
		$sudo_cmd pacman -Sy --noconfirm
		;;
	zypper)
		$sudo_cmd zypper refresh
		;;
	apk)
		$sudo_cmd apk update
		;;
	brew)
		brew update
		;;
	*)
		log_warn "Unknown package manager. Cannot update index."
		;;
	esac
}

# Install one or more canonical or native packages
pkg_install() {
	local mgr distro
	mgr="$(detect_pkg_mgr)"
	distro="$(detect_distro)"

	local -a to_install=()

	for canonical_or_native in "$@"; do
		local translated
		translated="$(pkg_translate "$canonical_or_native")"

		# Skip empty translations (e.g. Linux-only tools on macOS/Termux)
		if [[ -z "$translated" ]]; then
			log_debug "Package '${canonical_or_native}' skipped (not applicable on ${distro})."
			continue
		fi

		# Split space-separated translated package lists
		for native_pkg in $translated; do
			if [[ "$native_pkg" == @* ]]; then
				# Group package (e.g. @development-tools in dnf)
				to_install+=("$native_pkg")
			elif pkg_is_installed "$native_pkg"; then
				log_debug "Package already installed: ${native_pkg}"
			else
				to_install+=("$native_pkg")
			fi
		done
	done

	if ((${#to_install[@]} == 0)); then
		log_debug "All requested packages are already satisfied."
		return 0
	fi

	log_step "Installing native packages: ${BOLD_WHITE}${to_install[*]}${RESET}"

	local sudo_cmd="sudo"
	[[ $EUID -eq 0 ]] && sudo_cmd=""

	case "$mgr" in
	termux)
		pkg install -y "${to_install[@]}"
		;;
	apt)
		$sudo_cmd DEBIAN_FRONTEND=noninteractive apt install -y --no-install-recommends "${to_install[@]}"
		;;
	dnf)
		$sudo_cmd dnf install -y "${to_install[@]}"
		;;
	pacman)
		$sudo_cmd pacman -S --needed --noconfirm "${to_install[@]}"
		;;
	zypper)
		$sudo_cmd zypper --non-interactive install "${to_install[@]}"
		;;
	apk)
		$sudo_cmd apk add --no-cache "${to_install[@]}"
		;;
	brew)
		brew install "${to_install[@]}"
		;;
	*)
		log_fatal "Unsupported package manager: ${mgr}. Cannot install: ${to_install[*]}"
		;;
	esac

	log_success "Packages successfully installed: ${to_install[*]}"
}

# Remove one or more packages cleanly
pkg_remove() {
	local mgr
	mgr="$(detect_pkg_mgr)"

	local -a to_remove=()
	for pkg in "$@"; do
		local translated
		translated="$(pkg_translate "$pkg")"
		for native_pkg in $translated; do
			if pkg_is_installed "$native_pkg"; then
				to_remove+=("$native_pkg")
			fi
		done
	done

	if ((${#to_remove[@]} == 0)); then
		return 0
	fi

	log_step "Removing packages: ${BOLD_WHITE}${to_remove[*]}${RESET}"

	local sudo_cmd="sudo"
	[[ $EUID -eq 0 ]] && sudo_cmd=""

	case "$mgr" in
	termux)
		pkg uninstall -y "${to_remove[@]}"
		;;
	apt)
		$sudo_cmd apt remove -y --purge "${to_remove[@]}"
		;;
	dnf)
		$sudo_cmd dnf remove -y "${to_remove[@]}"
		;;
	pacman)
		$sudo_cmd pacman -Rns --noconfirm "${to_remove[@]}"
		;;
	zypper)
		$sudo_cmd zypper --non-interactive remove "${to_remove[@]}"
		;;
	apk)
		$sudo_cmd apk del "${to_remove[@]}"
		;;
	brew)
		brew uninstall "${to_remove[@]}"
		;;
	*)
		log_warn "Unsupported package manager: ${mgr}. Skipping removal."
		;;
	esac
}
