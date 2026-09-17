# 构建当前平台的插件可执行文件（方案 B）
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
} finally {
  Pop-Location
}
