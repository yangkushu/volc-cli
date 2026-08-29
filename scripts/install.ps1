# volc-cli Release 安装器(Windows): 默认安装 CLI；-WithSkill 同时安装 Agent Skill
# 用法: powershell -ExecutionPolicy Bypass -Command "irm https://raw.githubusercontent.com/yangkushu/volc-cli/master/scripts/install.ps1 | iex"
param(
    [switch]$WithSkill
)

$ErrorActionPreference = "Stop"

$repo = "yangkushu/volc-cli"
$api = "https://api.github.com/repos/$repo/releases/latest"
try {
    $release = Invoke-RestMethod -Uri $api
} catch {
    Write-Error "获取最新 release 失败: $_ 请检查 https://github.com/$repo/releases/latest 并重试"
    exit 1
}
$arch = if ($env:PROCESSOR_ARCHITEW6432) { $env:PROCESSOR_ARCHITEW6432 } else { $env:PROCESSOR_ARCHITECTURE }
switch -Regex ($arch) {
    "^(AMD64|x86_64)$" { $goArch = "amd64"; break }
    "^(ARM64|aarch64)$" { $goArch = "arm64"; break }
    default {
        Write-Error "不支持的 CPU 架构: $arch (仅支持 amd64/arm64)"
        exit 1
    }
}
$assetName = "volc-cli-windows-$goArch.exe"
$asset = $release.assets | Where-Object { $_.name -eq $assetName }
if (-not $asset) {
    Write-Error "未找到资产 $assetName, 请检查 https://github.com/$repo/releases/latest"
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

if ($WithSkill) {
    # 从与二进制相同的 Release tag 下载完整目录，避免 Skill 版本漂移。
    $stage = Join-Path ([System.IO.Path]::GetTempPath()) ("volc-cli-skill-" + [guid]::NewGuid())
    $archive = Join-Path $stage "source.zip"
    try {
        New-Item -ItemType Directory -Force -Path $stage | Out-Null
        Invoke-WebRequest -Uri "https://github.com/$repo/archive/refs/tags/$($release.tag_name).zip" -OutFile $archive -UseBasicParsing
        Expand-Archive -Path $archive -DestinationPath $stage -Force
        $skillFile = Get-ChildItem -Path $stage -Filter "SKILL.md" -File -Recurse |
            Where-Object { $_.FullName -match '[\\/]skills[\\/]volc-cli[\\/]SKILL\.md$' } |
            Select-Object -First 1
        if (-not $skillFile) { throw "Release $($release.tag_name) 中未找到 volc-cli skill" }
        foreach ($dest in @("$HOME\.claude\skills\volc-cli", "$HOME\.codex\skills\volc-cli", "$HOME\.cursor\skills\volc-cli")) {
            New-Item -ItemType Directory -Force -Path $dest | Out-Null
            Copy-Item -Path (Join-Path $skillFile.DirectoryName "*") -Destination $dest -Recurse -Force
            Write-Host "skill 已安装: $dest ($($release.tag_name))"
        }
    } catch {
        Write-Error "安装 Skill 失败: $_ 请检查 https://github.com/$repo/releases/latest 并重试"
        exit 1
    } finally {
        if (Test-Path $stage) { Remove-Item -Path $stage -Recurse -Force }
    }
}

Write-Host "✔ 安装完成: $installDir\volc-cli.exe ($($release.tag_name))"
Write-Host "  请将 $installDir 加入 PATH, 或使用完整路径调用"
Write-Host "  凭证配置: setx VOLC_ACCESSKEY ... / setx VOLC_SECRETKEY ..."
Write-Host "  验证: volc-cli check-credentials"
if (-not $WithSkill) {
    Write-Host "  如需安装 Agent Skill，请执行 README 中的 npx skills 命令，或以 -WithSkill 运行本脚本。"
}
