# Path to your .env file
$envFilePath = ".\.env"

# Check if the .env file exists
if (-Not (Test-Path $envFilePath)) {
    Write-Host "Error: .env file not found at $envFilePath" -ForegroundColor Red
    exit 1
}

# Read the .env file line by line
Get-Content $envFilePath | ForEach-Object {
    # Skip empty lines and comments (lines starting with #)
    if ($_ -match "^\s*#" -or $_ -match "^\s*$") {
        return
    }

    # Split each line into key-value pairs
    $key, $value = $_ -split "=", 2

    # Trim whitespace from key and value
    $key = $key.Trim()
    $value = $value.Trim()

    # Set the environment variable
    [System.Environment]::SetEnvironmentVariable($key, $value, "Process")

    # Optionally, print the exported variable
    Write-Host "Exported: $key=$value"
}

Write-Host "All variables from .env have been exported." -ForegroundColor Green