$services = @("frontend", "backend", "scanner", "database")

foreach ($service in $services) {

    $id = docker compose ps -q $service

    if (-not $id) {
        Write-Host "$service : Container nicht gefunden"
        continue
    }

    $container = (docker inspect $id | ConvertFrom-Json)[0]

    Write-Host ""
    Write-Host "==============================="
    Write-Host "Service: $service"
    Write-Host "==============================="

    Write-Host "Benutzer:" $container.Config.User
    Write-Host "Privileged:" $container.HostConfig.Privileged
    Write-Host "ReadOnlyRootfs:" $container.HostConfig.ReadonlyRootfs
    Write-Host "CapAdd:" $container.HostConfig.CapAdd
    Write-Host "CapDrop:" $container.HostConfig.CapDrop
    Write-Host "SecurityOpt:" $container.HostConfig.SecurityOpt
}