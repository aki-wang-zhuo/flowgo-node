# 构建当前平台的插件可执行文件，并复制节点文档 Markdown。
# 产出：dist/echo[.exe]、dist/pluginEcho_zh.md、dist/pluginEcho_en.md
# 可打包 zip：dist/echo.zip（含二进制 + 文档，安装时文档会复制到 data/docs）
$ErrorActionPreference = "Stop"
$outDir = Join-Path $PSScriptRoot "dist"
New-Item -ItemType Directory -Force -Path $outDir | Out-Null
$ext = ""
if ($IsWindows -or $env:OS -match "Windows") { $ext = ".exe" }
$out = Join-Path $outDir "echo$ext"
Push-Location $PSScriptRoot
try {
  go build -o $out .
  Write-Host "built: $out"
  Copy-Item -Force (Join-Path $PSScriptRoot "pluginEcho_zh.md") (Join-Path $outDir "pluginEcho_zh.md")
  Copy-Item -Force (Join-Path $PSScriptRoot "pluginEcho_en.md") (Join-Path $outDir "pluginEcho_en.md")
  Write-Host "copied docs to dist/"

  $zipPath = Join-Path $outDir "echo.zip"
  if (Test-Path $zipPath) { Remove-Item -Force $zipPath }
  Compress-Archive -Path $out, (Join-Path $outDir "pluginEcho_zh.md"), (Join-Path $outDir "pluginEcho_en.md") -DestinationPath $zipPath
  Write-Host "zip: $zipPath"
} finally {
  Pop-Location
}
