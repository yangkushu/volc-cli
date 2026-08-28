# volc-cli 一键安装(Windows): 下载最新 release + 安装 skills
# 用法: powershell -ExecutionPolicy Bypass -Command "irm https://raw.githubusercontent.com/yangkushu/volc-cli/master/scripts/install.ps1 | iex"
$ErrorActionPreference = "Stop"

$repo = "yangkushu/volc-cli"
$api = "https://api.github.com/repos/$repo/releases/latest"
try {
    $release = Invoke-RestMethod -Uri $api
} catch {
    Write-Error "获取最新 release 失败: $_ 请检查 https://github.com/$repo/releases/latest 并重试"
    exit 1
}
$asset = $release.assets | Where-Object { $_.name -eq "volc-cli-windows-amd64.exe" }
if (-not $asset) {
    Write-Error "未找到资产 volc-cli-windows-amd64.exe, 请检查 https://github.com/$repo/releases/latest"
    exit 1
}
$installDir = if ($env:VOLC_CLI_INSTALL_DIR) { $env:VOLC_CLI_INSTALL_DIR } else { "$env:LOCALAPPDATA\volc-cli" }
New-Item -ItemType Directory -Force -Path $installDir | Out-Null
try {
    $resp = Invoke-WebRequest -Uri $asset.browser_download_url -OutFile "$installDir\volc-cli.exe" -UseBasicParsing
} catch {
    Write-Error "下载 volc-cli 可执行文件失败: $_ 请检查 https://github.com/$repo/releases/latest 并重试"
    exit 1
}
if ($resp.StatusCode -ne 200) {
    Write-Error "下载 volc-cli 可执行文件失败(HTTP $($resp.StatusCode)): 请检查 https://github.com/$repo/releases/latest 并重试"
    exit 1
}

$skillsSrc = "https://raw.githubusercontent.com/$repo/master/skills/volc-cli/SKILL.md"
foreach ($dest in @("$HOME\.claude\skills\volc-cli", "$HOME\.codex\skills\volc-cli")) {
    New-Item -ItemType Directory -Force -Path $dest | Out-Null
    try {
        $resp = Invoke-WebRequest -Uri $skillsSrc -OutFile "$dest\SKILL.md" -UseBasicParsing
    } catch {
        Write-Error "下载 SKILL.md 失败: $_ 请检查 https://github.com/$repo/releases/latest 并重试"
        exit 1
    }
    if ($resp.StatusCode -ne 200) {
        Write-Error "下载 SKILL.md 失败(HTTP $($resp.StatusCode)): 请检查 https://github.com/$repo/releases/latest 并重试"
        exit 1
    }
    Write-Host "skill 已安装: $dest"
}

Write-Host "✔ 安装完成: $installDir\volc-cli.exe"
Write-Host "  请将 $installDir 加入 PATH, 或使用完整路径调用"
Write-Host "  凭证配置: setx VOLC_ACCESSKEY ... / setx VOLC_SECRETKEY ..."
Write-Host "  验证: volc-cli check-credentials"
