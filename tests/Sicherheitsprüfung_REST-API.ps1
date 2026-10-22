$tests = @(
    @{
        Name = "S01 - Oeffentliche IP"
        Body = '{"target":"8.8.8.8"}'
    },
    @{
        Name = "S02 - Zu grosses Netzwerk"
        Body = '{"target":"192.168.0.0/23"}'
    },
    @{
        Name = "S03 - IPv6"
        Body = '{"target":"::1"}'
    },
    @{
        Name = "S04 - Shell-Zeichen"
        Body = '{"target":"127.0.0.1; echo test"}'
    },
    @{
        Name = "S05 - Nmap-Parameter"
        Body = '{"target":"--script vuln"}'
    },
    @{
        Name = "S06 - Leeres Ziel"
        Body = '{"target":""}'
    },
    @{
        Name = "S07 - Unbekanntes JSON-Feld"
        Body = '{"target":"127.0.0.1","extra":"test"}'
    }
)

$passed = 0
$failed = 0

foreach ($test in $tests) {

    $statusCode = 0

    try {
        Invoke-RestMethod `
            -Uri "http://127.0.0.1:8080/api/scans" `
            -Method Post `
            -ContentType "application/json" `
            -Body $test.Body `
            -ErrorAction Stop | Out-Null

        $statusCode = 202
    }
    catch {
        if ($null -ne $_.Exception.Response) {
            $statusCode = [int]$_.Exception.Response.StatusCode
        }
    }

    if ($statusCode -eq 400) {
        Write-Host "BESTANDEN: $($test.Name)" -ForegroundColor Green
        $passed++
    }
    else {
        Write-Host "FEHLER: $($test.Name) - HTTP $statusCode" -ForegroundColor Red
        $failed++
    }
}

Write-Host ""
Write-Host "====================================="
Write-Host "SICHERHEITSTEST - ERGEBNIS"
Write-Host "Bestanden: $passed"
Write-Host "Fehlgeschlagen: $failed"
Write-Host "====================================="