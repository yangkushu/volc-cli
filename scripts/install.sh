#!/usr/bin/env bash
# volc-cli 一键安装: 下载最新 release 可执行文件 + 安装 Claude Code/Codex skills
# 用法: curl -fsSL https://raw.githubusercontent.com/yangkushu/volc-cli/master/scripts/install.sh | bash
set -euo pipefail

REPO="yangkushu/volc-cli"
BIN="volc-cli"
API="https://api.github.com/repos/${REPO}/releases/latest"

# 1. 平台检测(只支持 linux/windows x64)
os="$(uname -s)"
case "$os" in
  Linux)  asset="volc-cli-linux-amd64" ;;
  MINGW*|MSYS*|CYGWIN*) asset="volc-cli-windows-amd64.exe" ;;
  *) echo "不支持的平台: $os (仅支持 linux/windows x64)" >&2; exit 1 ;;
esac

# 2. 下载最新 release
url="$(curl -fsSL "$API" | sed -n "s/.*\"browser_download_url\": *\"\([^\"]*${asset}[^\"]*\)\".*/\1/p" | head -1)"
if [ -z "$url" ]; then
  echo "未找到资产 ${asset}, 请检查 https://github.com/${REPO}/releases/latest" >&2
  exit 1
fi
install_dir="${VOLC_CLI_INSTALL_DIR:-$HOME/bin}"
mkdir -p "$install_dir"
tmp="$(mktemp)"
curl -fSL -o "$tmp" "$url"
chmod +x "$tmp"
mv "$tmp" "$install_dir/$BIN"

# 3. 安装 skills(claude code + codex 共用同一份 SKILL.md)
skills_src="https://raw.githubusercontent.com/${REPO}/master/skills/volc-cli/SKILL.md"
for dest in "$HOME/.claude/skills/volc-cli" "$HOME/.codex/skills/volc-cli"; do
  mkdir -p "$dest"
  curl -fSL -o "$dest/SKILL.md" "$skills_src"
  echo "skill 已安装: $dest"
done

echo "✔ 安装完成: $install_dir/$BIN"
echo "  请确认 $install_dir 在 PATH 中, 或执行: export PATH=\"$install_dir:\$PATH\""
echo "  凭证配置: export VOLC_ACCESSKEY=... VOLC_SECRETKEY=..."
echo "  验证: volc-cli check-credentials"
