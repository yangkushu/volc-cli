#!/usr/bin/env bash
# volc-cli Release 安装器: 默认安装 CLI；--with-skill 同时安装 Agent Skill
# 用法: curl -fsSL https://raw.githubusercontent.com/yangkushu/volc-cli/master/scripts/install.sh | bash -s -- [--with-skill]
set -euo pipefail

REPO="yangkushu/volc-cli"
API="https://api.github.com/repos/${REPO}/releases/latest"

install_skill=0
case "${1:-}" in
  "") ;;
  --with-skill) install_skill=1 ;;
  -h|--help)
    echo "用法: install.sh [--with-skill]"
    echo "默认只安装 volc-cli；--with-skill 额外安装 Claude Code、Codex、Cursor Skill。"
    exit 0
    ;;
  *)
    echo "未知参数: $1（仅支持 --with-skill）" >&2
    exit 1
    ;;
esac

# 1. 平台与架构检测
os="$(uname -s)"
arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  aarch64|arm64) arch="arm64" ;;
  *) echo "不支持的 CPU 架构: $arch (仅支持 amd64/arm64)" >&2; exit 1 ;;
esac

case "$os" in
  Linux) platform="linux"; bin="volc-cli" ;;
  Darwin) platform="darwin"; bin="volc-cli" ;;
  MINGW*|MSYS*|CYGWIN*) platform="windows"; bin="volc-cli.exe" ;;
  *) echo "不支持的平台: $os (仅支持 Linux/macOS/Windows)" >&2; exit 1 ;;
esac
asset="volc-cli-${platform}-${arch}"
[ "$platform" = "windows" ] && asset="${asset}.exe"

# 2. 下载最新 release；Skill 从同一 tag 的源码目录提取，避免与二进制版本漂移
release_json="$(curl -fsSL "$API")"
url="$(printf '%s' "$release_json" | sed -n "s/.*\"browser_download_url\": *\"\([^\"]*\/${asset}\)\".*/\1/p" | head -1)"
tag="$(printf '%s' "$release_json" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -1)"
if [ -z "$url" ]; then
  echo "未找到资产 ${asset}, 请检查 https://github.com/${REPO}/releases/latest" >&2
  exit 1
fi
if [ -z "$tag" ]; then
  echo "无法解析最新 Release 的 tag, 请检查 https://github.com/${REPO}/releases/latest" >&2
  exit 1
fi
install_dir="${VOLC_CLI_INSTALL_DIR:-$HOME/bin}"
mkdir -p "$install_dir"
tmp="$(mktemp)"
cleanup() { rm -f "$tmp"; }
trap cleanup EXIT
curl -fSL -o "$tmp" "$url"
chmod +x "$tmp"
mv "$tmp" "$install_dir/$bin"

# 3. 可选：安装完整 Skill 目录。
if [ "$install_skill" = "1" ]; then
  skill_stage="$(mktemp -d)"
  cleanup() { rm -f "$tmp"; rm -rf "$skill_stage"; }
  curl -fsSL "https://github.com/${REPO}/archive/refs/tags/${tag}.tar.gz" | tar -xzf - -C "$skill_stage"
  skill_file="$(find "$skill_stage" -type f -path '*/skills/volc-cli/SKILL.md' -print -quit)"
  if [ -z "$skill_file" ]; then
    echo "Release ${tag} 中未找到 volc-cli skill" >&2
    exit 1
  fi
  skill_src="${skill_file%/SKILL.md}"
  for dest in "$HOME/.claude/skills/volc-cli" "$HOME/.codex/skills/volc-cli" "$HOME/.cursor/skills/volc-cli"; do
    mkdir -p "$dest"
    cp -a "$skill_src"/. "$dest"/
    echo "skill 已安装: $dest (${tag})"
  done
fi

echo "✔ 安装完成: $install_dir/$bin (${tag})"
echo "  请确认 $install_dir 在 PATH 中, 或执行: export PATH=\"$install_dir:\$PATH\""
echo "  凭证配置: export VOLC_ACCESSKEY=... VOLC_SECRETKEY=..."
echo "  验证: volc-cli check-credentials"
if [ "$install_skill" = "0" ]; then
  echo "  如需安装 Agent Skill，请执行 README 中的 npx skills 命令，或重跑本脚本并传入 --with-skill。"
fi
