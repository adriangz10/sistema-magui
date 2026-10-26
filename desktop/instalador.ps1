<#
.SYNOPSIS
  Genera el instalador de Magui para Windows (build\bin\Magui-amd64-installer.exe).

.DESCRIPTION
  Compila la app de escritorio apuntando al servidor indicado y arma el instalador con NSIS.
  Se instala para el usuario actual (sin pedir permisos de administrador) en %LOCALAPPDATA%\Programs\Magui.
  La versión queda en wails.json (se ve en "Aplicaciones instaladas") y en magui.log.

.EXAMPLE
  .\instalador.ps1 -Servidor https://magui.ejemplo.com -Version 1.0.0
#>
param(
    [Parameter(Mandatory = $true)][string]$Servidor,
    [Parameter(Mandatory = $true)][string]$Version
)
$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot

$uri = $null
if (-not [Uri]::TryCreate($Servidor.TrimEnd('/'), [UriKind]::Absolute, [ref]$uri) -or $uri.Scheme -notin @('http', 'https')) {
    throw "Servidor inválido: '$Servidor'. Use por ejemplo https://magui.ejemplo.com"
}
if ($uri.Scheme -ne 'https' -and $uri.Host -notin @('localhost', '127.0.0.1')) {
    throw "El servidor de producción debe usar https:// (la contraseña viaja por la red)."
}
if ($Version -notmatch '^\d+\.\d+\.\d+$') {
    throw "Versión inválida: '$Version'. Use tres números, por ejemplo 1.0.0"
}

# makensis viene con NSIS (winget install NSIS.NSIS)
$nsis = 'C:\Program Files (x86)\NSIS'
if (-not (Get-Command makensis -ErrorAction SilentlyContinue) -and (Test-Path "$nsis\makensis.exe")) {
    $env:PATH = "$env:PATH;$nsis"
}
if (-not (Get-Command makensis -ErrorAction SilentlyContinue)) { throw 'Falta NSIS: winget install NSIS.NSIS' }
if (-not (Get-Command wails -ErrorAction SilentlyContinue)) { throw 'Falta Wails: go install github.com/wailsapp/wails/v2/cmd/wails@latest' }

# Versión en wails.json (la usa el instalador)
$cfg = Get-Content wails.json -Raw | ConvertFrom-Json
$cfg.info.productVersion = $Version
$cfg | ConvertTo-Json -Depth 10 | Set-Content wails.json -Encoding utf8NoBOM

$servidorFinal = $uri.GetLeftPart([UriPartial]::Authority)
wails build -clean -skipbindings -nsis -installscope user -ldflags "-X main.servidorPorDefecto=$servidorFinal -X main.version=$Version"
if ($LASTEXITCODE -ne 0) { throw 'Falló la compilación' }

$instalador = Join-Path $PSScriptRoot 'build\bin\Magui-amd64-installer.exe'
Write-Host ''
Write-Host "Instalador listo: $instalador"
Write-Host "Servidor: $servidorFinal  Versión: $Version"
