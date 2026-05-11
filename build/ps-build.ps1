param(
    [string]$appName = "gmail-planner",
    [parameter(Mandatory=$true)][string]$version = "1.0.0",
    [string]$rootPath = "C:\0_Priv\github.com\gmail-planner"
)

$startPath = (Get-Location).path

$env:GOOS="windows"
$env:GOARCH="amd64"

Set-location "$($rootPath)\src"
go build -ldflags="-s -w -X main.version=$($version)" -o "$($rootPath)\build\bin\$($appName)_$($version).exe" .\cmd

$env:GOOS="linux"
$env:GOARCH="amd64"
go build -ldflags="-s -w -X main.version=$($version)" -o "$($rootPath)\build\bin\$($appName)_$($version)_linux_amd64" .\cmd

$env:GOOS="windows"
Set-location $startPath