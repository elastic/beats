# Specific GUID for "Active Directory Domain Services: SAM" (or related security events)
$providerGuid = "{68FDD900-4A3E-11D1-84F4-0000F80464E3}"

try {
    # Get the provider information using the GUID
    $providerInfo = Get-WinEvent -ListProvider $providerGuid -ErrorAction Stop

    # Check if the provider has any keywords defined
    if ($providerInfo.Keywords.Count -gt 0) {
        Write-Host "Keywords for provider: $($providerInfo.Name) (GUID: $providerGuid)"
        Write-Host "--------------------------------------------------"
        
        # Loop through each keyword and display its name and mask value
        foreach ($keyword in $providerInfo.Keywords) {
            # The 'Value' property holds the keyword mask (long/int64)
            # Format it as hexadecimal for common representation
            $maskHex = "0x{0:X}" -f $keyword.Value
            Write-Host ("Name: {0,-45} Mask: {1,-20} Value (Decimal): {2}" -f $keyword.Name, $maskHex, $keyword.Value)
        }
    }
    else {
        Write-Host "No keywords found or defined for provider: $($providerInfo.Name) (GUID: $providerGuid)"
    }
}
catch {
    # Handle errors, e.g., if the provider is not found on the system
    Write-Error "Could not retrieve information for provider GUID $providerGuid : $($_.Exception.Message)"
}