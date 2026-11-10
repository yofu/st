param([Parameter(Mandatory=$true)][string]$OutDir)
$ErrorActionPreference = 'Stop'
$destination = [IO.Path]::GetFullPath($OutDir)
New-Item -ItemType Directory -Force -Path $destination | Out-Null
Push-Location (Join-Path $PSScriptRoot '../..')
try {
    $prevOS, $prevArch, $prevCgo = $env:GOOS, $env:GOARCH, $env:CGO_ENABLED
    $env:GOOS = 'js'; $env:GOARCH = 'wasm'; $env:CGO_ENABLED = '0'
    go build -buildvcs=false -trimpath '-ldflags=-s -w' -o (Join-Path $destination 'stio.wasm') ./cmd/stioweb
    if ($LASTEXITCODE -ne 0) { throw 'WebAssembly build failed.' }
    go build -buildvcs=false -trimpath '-ldflags=-s -w' -o (Join-Path $destination 'analysis.wasm') ./cmd/stiowebworker
    if ($LASTEXITCODE -ne 0) { throw 'Analysis WebAssembly build failed.' }
    $goRoot = go env GOROOT
    if ($LASTEXITCODE -ne 0) { throw 'GOROOT is unavailable.' }
    Copy-Item -LiteralPath (Join-Path $goRoot 'lib/wasm/wasm_exec.js') -Destination $destination
    Copy-Item -LiteralPath (Join-Path $goRoot 'LICENSE') -Destination (Join-Path $destination 'GO-LICENSE.txt')
    foreach ($name in 'index.html','viewer.css','viewer.js','workspace.js','analysis-worker.js') { Copy-Item -LiteralPath (Join-Path $PSScriptRoot $name) -Destination $destination }
} finally {
    $env:GOOS = $prevOS; $env:GOARCH = $prevArch; $env:CGO_ENABLED = $prevCgo
    Pop-Location
}
