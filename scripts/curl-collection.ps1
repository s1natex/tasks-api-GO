param(
  [string]$BaseUrl = "http://localhost:8080",
  [string]$HealthUrl = "http://localhost:8081"
)

Write-Host "# health (app)"
Invoke-WebRequest -Uri "$BaseUrl/health" -Method GET | Select-Object -ExpandProperty Content

Write-Host "`n# create task"
(Invoke-WebRequest -Uri "$BaseUrl/tasks" -Method POST -ContentType "application/json" -Body '{"title":"from powershell"}').Content

Write-Host "`n# create task (422)"
try {
  (Invoke-WebRequest -Uri "$BaseUrl/tasks" -Method POST -ContentType "application/json" -Body '{"title":""}').Content
} catch {
  $_.Exception.Response.GetResponseStream() | % { $reader = New-Object System.IO.StreamReader($_); $reader.ReadToEnd() }
}

Write-Host "`n# list tasks"
Invoke-WebRequest -Uri "$BaseUrl/tasks" -Method GET | Select-Object -ExpandProperty Content

Write-Host "`n# metrics (first 10 lines)"
$metrics = Invoke-WebRequest -Uri "$BaseUrl/metrics" -Method GET | Select-Object -ExpandProperty Content
$metrics -split "`n" | Select-Object -First 10 | ForEach-Object { $_ }

Write-Host "`n# cors preflight"
Invoke-WebRequest -Uri "$BaseUrl/tasks" -Method OPTIONS -Headers @{
  "Origin"="https://example.com"
  "Access-Control-Request-Method"="POST"
} | Out-Null
"OK"

Write-Host "`n# health (drain)"
Invoke-WebRequest -Uri "$HealthUrl/health" -Method GET | Select-Object -ExpandProperty Content
