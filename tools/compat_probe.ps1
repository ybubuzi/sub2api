param(
  [Parameter(Mandatory = $true)]
  [string]$BaseUrl,

  [Parameter(Mandatory = $true)]
  [string]$ClaudeApiKey,

  [Parameter(Mandatory = $true)]
  [string]$CodexApiKey,

  [string]$ClaudeModel = "claude-sonnet-4-6",
[string]$CodexModel = "gpt-5.3-codex",
  [int]$TimeoutSec = 120,
  [switch]$Stream
)

$ErrorActionPreference = "Stop"

function Join-UrlPath {
  param(
    [Parameter(Mandatory = $true)][string]$Root,
    [Parameter(Mandatory = $true)][string]$Path
  )
  return $Root.TrimEnd('/') + '/' + $Path.TrimStart('/')
}

function ConvertTo-PrettyJson {
  param([Parameter(Mandatory = $true)]$Value)
  return $Value | ConvertTo-Json -Depth 32 -Compress:$false
}

function New-CompatResult {
  param(
    [Parameter(Mandatory = $true)][string]$Name,
    [Parameter(Mandatory = $true)][string]$Endpoint,
    [Parameter(Mandatory = $true)][int]$StatusCode,
    [string]$ResponseType,
    [string]$ResponseId,
    [string]$Model,
    [string]$Preview,
    [bool]$Passed,
    [string]$ErrorMessage
  )
  [pscustomobject]@{
    name = $Name
    endpoint = $Endpoint
    status_code = $StatusCode
    response_type = $ResponseType
    response_id = $ResponseId
    model = $Model
    preview = $Preview
    passed = $Passed
    error = $ErrorMessage
  }
}

function Invoke-CompatJsonRequest {
  param(
    [Parameter(Mandatory = $true)][string]$Name,
    [Parameter(Mandatory = $true)][string]$Path,
    [Parameter(Mandatory = $true)][string]$ApiKey,
    [Parameter(Mandatory = $true)]$Body,
    [Parameter(Mandatory = $true)][scriptblock]$Validate
  )

  $endpoint = Join-UrlPath -Root $BaseUrl -Path $Path
  $headers = @{
    Authorization = "Bearer $ApiKey"
    "Content-Type" = "application/json"
    Accept = "application/json"
  }
  $jsonBody = ConvertTo-PrettyJson -Value $Body

  try {
    $response = Invoke-WebRequest -Method Post -Uri $endpoint -Headers $headers -Body $jsonBody -TimeoutSec $TimeoutSec -SkipHttpErrorCheck
    $raw = [string]$response.Content
    $parsed = $null
    if ($raw.Trim().Length -gt 0) {
      try { $parsed = $raw | ConvertFrom-Json -Depth 32 } catch { }
    }

    $validation = & $Validate $response.StatusCode $parsed $raw
    $preview = if ($raw.Length -gt 600) { $raw.Substring(0, 600) } else { $raw }
    return New-CompatResult `
      -Name $Name `
      -Endpoint $endpoint `
      -StatusCode ([int]$response.StatusCode) `
      -ResponseType $(if ($null -ne $parsed -and $parsed.PSObject.Properties.Name -contains "type") { [string]$parsed.type } else { "" }) `
      -ResponseId $(if ($null -ne $parsed -and $parsed.PSObject.Properties.Name -contains "id") { [string]$parsed.id } else { "" }) `
      -Model $(if ($null -ne $parsed -and $parsed.PSObject.Properties.Name -contains "model") { [string]$parsed.model } else { "" }) `
      -Preview $preview `
      -Passed ([bool]$validation.passed) `
      -ErrorMessage ([string]$validation.error)
  } catch {
    return New-CompatResult -Name $Name -Endpoint $endpoint -StatusCode 0 -Passed $false -ErrorMessage $_.Exception.Message
  }
}

function Invoke-CompatStreamRequest {
  param(
    [Parameter(Mandatory = $true)][string]$Name,
    [Parameter(Mandatory = $true)][string]$Path,
    [Parameter(Mandatory = $true)][string]$ApiKey,
    [Parameter(Mandatory = $true)]$Body,
    [Parameter(Mandatory = $true)][string[]]$ExpectedMarkers
  )

  $endpoint = Join-UrlPath -Root $BaseUrl -Path $Path
  $headers = @{
    Authorization = "Bearer $ApiKey"
    "Content-Type" = "application/json"
    Accept = "text/event-stream"
  }
  $jsonBody = ConvertTo-PrettyJson -Value $Body

  try {
    $response = Invoke-WebRequest -Method Post -Uri $endpoint -Headers $headers -Body $jsonBody -TimeoutSec $TimeoutSec -SkipHttpErrorCheck
    $raw = [string]$response.Content
    $passed = [int]$response.StatusCode -ge 200 -and [int]$response.StatusCode -lt 300
    foreach ($marker in $ExpectedMarkers) {
      if (-not $raw.Contains($marker)) { $passed = $false }
    }
    $preview = if ($raw.Length -gt 900) { $raw.Substring(0, 900) } else { $raw }
    $missing = @($ExpectedMarkers | Where-Object { -not $raw.Contains($_) })
    $err = if ($missing.Count -gt 0) { "missing stream marker(s): " + ($missing -join ", ") } else { "" }
    return New-CompatResult -Name $Name -Endpoint $endpoint -StatusCode ([int]$response.StatusCode) -Preview $preview -Passed $passed -ErrorMessage $err
  } catch {
    return New-CompatResult -Name $Name -Endpoint $endpoint -StatusCode 0 -Passed $false -ErrorMessage $_.Exception.Message
  }
}

$results = New-Object System.Collections.Generic.List[object]

$openAIResponsesBody = @{
  model = $ClaudeModel
  input = @(
    @{
      role = "user"
      content = @(
        @{ type = "input_text"; text = "Reply with exactly: compat-ok" }
      )
    }
  )
  max_output_tokens = 32
  stream = [bool]$Stream
}

$openAIChatBody = @{
  model = $ClaudeModel
  messages = @(
    @{ role = "user"; content = "Reply with exactly: compat-ok" }
  )
  max_tokens = 32
  stream = [bool]$Stream
}

$anthropicMessagesBody = @{
  model = $CodexModel
  max_tokens = 32
  messages = @(
    @{ role = "user"; content = "Reply with exactly: compat-ok" }
  )
  stream = [bool]$Stream
}

if ($Stream) {
  $results.Add((Invoke-CompatStreamRequest -Name "claude-platform accepts OpenAI Responses" -Path "/v1/responses" -ApiKey $ClaudeApiKey -Body $openAIResponsesBody -ExpectedMarkers @("event:", "response.")))
  $results.Add((Invoke-CompatStreamRequest -Name "claude-platform accepts OpenAI Chat Completions" -Path "/v1/chat/completions" -ApiKey $ClaudeApiKey -Body $openAIChatBody -ExpectedMarkers @("data:")))
  $results.Add((Invoke-CompatStreamRequest -Name "codex-platform accepts Claude Messages" -Path "/v1/messages" -ApiKey $CodexApiKey -Body $anthropicMessagesBody -ExpectedMarkers @("event:", "message_")))
} else {
  $results.Add((Invoke-CompatJsonRequest -Name "claude-platform accepts OpenAI Responses" -Path "/v1/responses" -ApiKey $ClaudeApiKey -Body $openAIResponsesBody -Validate {
    param($status, $parsed, $raw)
    $ok = $status -ge 200 -and $status -lt 300 -and $null -ne $parsed -and $parsed.PSObject.Properties.Name -contains "id" -and $parsed.PSObject.Properties.Name -contains "output"
    [pscustomobject]@{ passed = $ok; error = $(if ($ok) { "" } else { "expected OpenAI Responses JSON with id/output" }) }
  }))
  $results.Add((Invoke-CompatJsonRequest -Name "claude-platform accepts OpenAI Chat Completions" -Path "/v1/chat/completions" -ApiKey $ClaudeApiKey -Body $openAIChatBody -Validate {
    param($status, $parsed, $raw)
    $ok = $status -ge 200 -and $status -lt 300 -and $null -ne $parsed -and $parsed.PSObject.Properties.Name -contains "choices"
    [pscustomobject]@{ passed = $ok; error = $(if ($ok) { "" } else { "expected Chat Completions JSON with choices" }) }
  }))
  $results.Add((Invoke-CompatJsonRequest -Name "codex-platform accepts Claude Messages" -Path "/v1/messages" -ApiKey $CodexApiKey -Body $anthropicMessagesBody -Validate {
    param($status, $parsed, $raw)
    $ok = $status -ge 200 -and $status -lt 300 -and $null -ne $parsed -and $parsed.type -eq "message" -and $parsed.PSObject.Properties.Name -contains "content"
    [pscustomobject]@{ passed = $ok; error = $(if ($ok) { "" } else { "expected Anthropic Messages JSON with type=message/content" }) }
  }))
}

$results | Format-Table -AutoSize name, status_code, passed, error
""
"--- Cross-Platform Fallback Tests ---"
"NOTE: These tests require the Claude group's fallback_group_id to point to an OpenAI group,"
"and the Codex group's fallback_group_id to point to a Claude group."
"If not configured, these tests will return 503 (expected when no fallback is set up)."
""

$crossPlatformClaudeBody = @{
  model = $ClaudeModel
  max_tokens = 32
  messages = @(
    @{ role = "user"; content = "Reply with exactly: cross-platform-ok" }
  )
  stream = [bool]$Stream
}

$crossPlatformCodexBody = @{
  model = $CodexModel
  input = @(
    @{
      role = "user"
      content = @(
        @{ type = "input_text"; text = "Reply with exactly: cross-platform-ok" }
      )
    }
  )
  max_output_tokens = 32
  stream = [bool]$Stream
}

if ($Stream) {
  $results.Add((Invoke-CompatStreamRequest -Name "claude-key cross-platform fallback to codex (/v1/messages)" -Path "/v1/messages" -ApiKey $ClaudeApiKey -Body $crossPlatformClaudeBody -ExpectedMarkers @("event:", "message_")))
  $results.Add((Invoke-CompatStreamRequest -Name "codex-key cross-platform fallback to claude (/v1/responses)" -Path "/v1/responses" -ApiKey $CodexApiKey -Body $crossPlatformCodexBody -ExpectedMarkers @("event:", "response.")))
} else {
  $results.Add((Invoke-CompatJsonRequest -Name "claude-key cross-platform fallback to codex (/v1/messages)" -Path "/v1/messages" -ApiKey $ClaudeApiKey -Body $crossPlatformClaudeBody -Validate {
    param($status, $parsed, $raw)
    $ok = $status -ge 200 -and $status -lt 300 -and $null -ne $parsed -and $parsed.type -eq "message" -and $parsed.PSObject.Properties.Name -contains "content"
    [pscustomobject]@{ passed = $ok; error = $(if ($ok) { "" } else { "expected Anthropic Messages JSON (forwarded via OpenAI upstream)" }) }
  }))
  $results.Add((Invoke-CompatJsonRequest -Name "codex-key cross-platform fallback to claude (/v1/responses)" -Path "/v1/responses" -ApiKey $CodexApiKey -Body $crossPlatformCodexBody -Validate {
    param($status, $parsed, $raw)
    $ok = $status -ge 200 -and $status -lt 300 -and $null -ne $parsed -and $parsed.PSObject.Properties.Name -contains "id" -and $parsed.PSObject.Properties.Name -contains "output"
    [pscustomobject]@{ passed = $ok; error = $(if ($ok) { "" } else { "expected OpenAI Responses JSON (forwarded via Anthropic upstream)" }) }
  }))
}

""
"Full result JSON:"
$results | ConvertTo-Json -Depth 32

if (($results | Where-Object { -not $_.passed }).Count -gt 0) {
  exit 1
}
