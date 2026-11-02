#!/usr/bin/env bash
# ==============================================================================
# install_pyenv.sh
# Resilient pyenv installer and working tree repair script for Linux and macOS.
# Part of arturonavax/env development environment.
# ==============================================================================

set -euo pipefail

PYENV_ROOT="${HOME}/.pyenv"
BIN_DIR="${HOME}/.local/bin"

mkdir -p "${BIN_DIR}"

if [ -d "${PYENV_ROOT}" ]; then
	if [ -x "${PYENV_ROOT}/bin/pyenv" ]; then
		echo "✓ pyenv already present at ${PYENV_ROOT}/bin/pyenv"
		ln -sf "${PYENV_ROOT}/bin/pyenv" "${BIN_DIR}/pyenv"
		exit 0
	fi

	# If git repo exists, attempt to restore missing files
	if [ -d "${PYENV_ROOT}/.git" ]; then
		echo "Attempting to restore pyenv git working tree..."
		git -C "${PYENV_ROOT}" reset --hard HEAD 2>/dev/null || true
		git -C "${PYENV_ROOT}" checkout -f master 2>/dev/null || true
		if [ -x "${PYENV_ROOT}/bin/pyenv" ]; then
			echo "✓ pyenv restored successfully at ${PYENV_ROOT}/bin/pyenv"
			ln -sf "${PYENV_ROOT}/bin/pyenv" "${BIN_DIR}/pyenv"
			exit 0
		fi
	fi

	# pyenv.run hard-fails if ~/.pyenv exists; clean up broken/incomplete directory
	echo "Removing incomplete or corrupted pyenv directory at ${PYENV_ROOT}..."
	rm -rf "${PYENV_ROOT}"
fi

echo "Installing pyenv via official installer..."
if command -v curl >/dev/null 2>&1; then
	curl -fsSL https://pyenv.run | bash
elif command -v wget >/dev/null 2>&1; then
	wget -qO- https://pyenv.run | bash
else
	echo "Error: Neither curl nor wget is available." >&2
	exit 1
fi

if [ -x "${PYENV_ROOT}/bin/pyenv" ]; then
	ln -sf "${PYENV_ROOT}/bin/pyenv" "${BIN_DIR}/pyenv"
	echo "✓ pyenv installed successfully to ${PYENV_ROOT}/bin/pyenv"
else
	echo "Error: pyenv binary not found at ${PYENV_ROOT}/bin/pyenv after install" >&2
	exit 1
fi
