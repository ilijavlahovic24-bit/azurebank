# AzureBank — end-to-end test script (PowerShell)
# Usage:  .\test.ps1
# Requires: PowerShell 5.1+ (default on Windows 10/11)

$ErrorActionPreference = "Stop"

$Base     = "http://localhost:8080"
$Email    = "test@test.com"
$Password = "secret123"

# --- helpers ---
function Pass($msg) { Write-Host "[OK] $msg" -ForegroundColor Green }
function Fail($msg) { Write-Host "[FAIL] $msg" -ForegroundColor Red; exit 1 }
function Info($msg) { Write-Host "--> $msg" -ForegroundColor Yellow }

# --- HTTP helper: vraca @{ Status = int; Body = obj/string } ---
function Invoke-Api {
    param(
        [string]$Method = "GET",
        [string]$Url,
        [string]$Token = $null,
        $Body = $null
    )

    $headers = @{}
    if ($Token) { $headers["Authorization"] = "Bearer $Token" }

    $params = @{
        Method  = $Method
        Uri     = $Url
        Headers = $headers
    }

    if ($Body -ne $null) {
        $params["Body"]        = ($Body | ConvertTo-Json -Compress)
        $params["ContentType"] = "application/json"
    }

    try {
        $resp = Invoke-WebRequest @params -UseBasicParsing
        $status = [int]$resp.StatusCode
        $parsed = $null
        try { $parsed = $resp.Content | ConvertFrom-Json } catch { $parsed = $resp.Content }
        return @{ Status = $status; Body = $parsed }
    }
    catch {
        # 4xx/5xx idu u catch kod Invoke-WebRequest
        $status = [int]$_.Exception.Response.StatusCode
        $body   = $null
        try {
            $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
            $raw = $reader.ReadToEnd()
            $reader.Close()
            try { $body = $raw | ConvertFrom-Json } catch { $body = $raw }
        } catch { $body = $null }
        return @{ Status = $status; Body = $body }
    }
}

Write-Host "=== AzureBank E2E test ===" -ForegroundColor Cyan
Write-Host ""

# ---------------------------------------------------------------
# 0. Health
# ---------------------------------------------------------------
Info "0. Health check"
$r = Invoke-Api -Url "$Base/health"
if ($r.Status -eq 200) { Pass "GET /health -> 200" } else { Fail "GET /health -> $($r.Status) (expected 200)" }

# ---------------------------------------------------------------
# 1. Register / Login
# ---------------------------------------------------------------
Info "1. Register / Login"
$r = Invoke-Api -Method POST -Url "$Base/auth/register" -Body @{ email = $Email; password = $Password }

if ($r.Status -eq 201) {
    Pass "POST /auth/register -> 201"
} elseif ($r.Status -eq 409) {
    Pass "POST /auth/register -> 409 (user already exists, continuing with login)"
} else {
    Fail "POST /auth/register -> $($r.Status)"
}

$r = Invoke-Api -Method POST -Url "$Base/auth/login" -Body @{ email = $Email; password = $Password }
$Token = $r.Body.token
if (-not $Token -or $Token -eq $null) { Fail "Login did not return a token" }
Pass "POST /auth/login -> token ($($Token.Substring(0,20))...)"

# ---------------------------------------------------------------
# 2. Auth error cases
# ---------------------------------------------------------------
Info "2. Auth error cases"

$r = Invoke-Api -Url "$Base/accounts"
if ($r.Status -eq 401) { Pass "GET /accounts without token -> 401" } else { Fail "GET /accounts without token -> $($r.Status)" }

$r = Invoke-Api -Method POST -Url "$Base/auth/login" -Body @{ email = $Email; password = "wrong" }
if ($r.Status -eq 401) { Pass "Login with wrong password -> 401" } else { Fail "Login with wrong password -> $($r.Status)" }

# ---------------------------------------------------------------
# 3. Accounts setup
# ---------------------------------------------------------------
Info "3. Setup accounts (checking + savings)"

$r = Invoke-Api -Url "$Base/accounts" -Token $Token
$accounts = @($r.Body)
$count = $accounts.Count

if ($count -lt 2) {
    $r = Invoke-Api -Method POST -Url "$Base/accounts" -Token $Token -Body @{ type = "checking" }
    $checkingId = $r.Body.id
    Pass "Created checking account (id=$checkingId)"

    $r = Invoke-Api -Method POST -Url "$Base/accounts" -Token $Token -Body @{ type = "savings" }
    $savingsId = $r.Body.id
    Pass "Created savings account (id=$savingsId)"
} else {
    $checkingId = ($accounts | Where-Object { $_.type -eq "checking" } | Select-Object -First 1).id
    $savingsId  = ($accounts | Where-Object { $_.type -eq "savings"  } | Select-Object -First 1).id
    Pass "Existing accounts: checking=$checkingId, savings=$savingsId"
}

$r = Invoke-Api -Method POST -Url "$Base/accounts" -Token $Token -Body @{ type = "crypto" }
if ($r.Status -eq 400) { Pass "POST /accounts with invalid type -> 400" } else { Fail "Invalid type -> $($r.Status)" }

# ---------------------------------------------------------------
# 4. Deposit
# ---------------------------------------------------------------
Info "4. Deposit"

$r = Invoke-Api -Method POST -Url "$Base/transactions/deposit" -Token $Token -Body @{ account_id = $checkingId; amount = 1000 }
if ($r.Status -eq 201) { Pass "Deposit 1000 to checking -> 201" } else { Fail "Deposit -> $($r.Status)" }

$r = Invoke-Api -Method POST -Url "$Base/transactions/deposit" -Token $Token -Body @{ account_id = $checkingId; amount = -50 }
if ($r.Status -eq 400) { Pass "Deposit with negative amount -> 400" } else { Fail "Negative deposit -> $($r.Status)" }

# ---------------------------------------------------------------
# 5. Withdraw
# ---------------------------------------------------------------
Info "5. Withdraw"

$r = Invoke-Api -Method POST -Url "$Base/transactions/withdraw" -Token $Token -Body @{ account_id = $checkingId; amount = 300 }
if ($r.Status -eq 201) { Pass "Withdraw 300 -> 201" } else { Fail "Withdraw -> $($r.Status)" }

$r = Invoke-Api -Method POST -Url "$Base/transactions/withdraw" -Token $Token -Body @{ account_id = $checkingId; amount = 99999 }
if ($r.Status -eq 422) { Pass "Withdraw above balance -> 422" } else { Fail "Withdraw above balance -> $($r.Status)" }

# ---------------------------------------------------------------
# 6. Transfer (ACID)
# ---------------------------------------------------------------
Info "6. Transfer"

$r = Invoke-Api -Url "$Base/accounts" -Token $Token
$beforeChecking = ($r.Body | Where-Object { $_.id -eq $checkingId } | Select-Object -First 1).balance
$beforeSavings  = ($r.Body | Where-Object { $_.id -eq $savingsId  } | Select-Object -First 1).balance
Info "Before transfer: checking=$beforeChecking, savings=$beforeSavings"

$r = Invoke-Api -Method POST -Url "$Base/transactions/transfer" -Token $Token -Body @{ from_account_id = $checkingId; to_account_id = $savingsId; amount = 200 }
if ($r.Status -eq 201) { Pass "Transfer 200 checking->savings -> 201" } else { Fail "Transfer -> $($r.Status)" }

$r = Invoke-Api -Url "$Base/accounts" -Token $Token
$afterChecking = ($r.Body | Where-Object { $_.id -eq $checkingId } | Select-Object -First 1).balance
$afterSavings  = ($r.Body | Where-Object { $_.id -eq $savingsId  } | Select-Object -First 1).balance
Info "After transfer: checking=$afterChecking, savings=$afterSavings"

# ACID: sum unchanged
if (($beforeChecking + $beforeSavings) -eq ($afterChecking + $afterSavings)) {
    Pass "ACID: total balance sum unchanged"
} else {
    Fail "ACID: sum changed! before=$($beforeChecking + $beforeSavings) after=$($afterChecking + $afterSavings)"
}

# individual deltas
if (($afterChecking -eq ($beforeChecking - 200)) -and ($afterSavings -eq ($beforeSavings + 200))) {
    Pass "ACID: checking -200, savings +200"
} else {
    Fail "ACID: individual balances incorrect"
}

# transfer to same account
$r = Invoke-Api -Method POST -Url "$Base/transactions/transfer" -Token $Token -Body @{ from_account_id = $checkingId; to_account_id = $checkingId; amount = 100 }
if ($r.Status -eq 400) { Pass "Transfer to same account -> 400" } else { Fail "Transfer to same account -> $($r.Status)" }

# transfer to non-existent
$r = Invoke-Api -Method POST -Url "$Base/transactions/transfer" -Token $Token -Body @{ from_account_id = $checkingId; to_account_id = 99999; amount = 100 }
if ($r.Status -eq 404) { Pass "Transfer to non-existent account -> 404" } else { Fail "Transfer to non-existent -> $($r.Status)" }

# transfer above balance
$r = Invoke-Api -Method POST -Url "$Base/transactions/transfer" -Token $Token -Body @{ from_account_id = $checkingId; to_account_id = $savingsId; amount = 99999 }
if ($r.Status -eq 422) { Pass "Transfer above balance -> 422" } else { Fail "Transfer above balance -> $($r.Status)" }

# ---------------------------------------------------------------
# 7. Transactions
# ---------------------------------------------------------------
Info "7. Transactions"

$r = Invoke-Api -Url "$Base/accounts/$checkingId/transactions" -Token $Token
$txCount = @($r.Body).Count
if ($txCount -ge 3) { Pass "GET /accounts/$checkingId/transactions -> $txCount transactions" } else { Fail "Transactions -> $txCount" }

$r = Invoke-Api -Url "$Base/accounts/99999/transactions" -Token $Token
if ($r.Status -eq 404) { Pass "Foreign/non-existent account -> 404" } else { Fail "Foreign account -> $($r.Status)" }

# ---------------------------------------------------------------
# Summary
# ---------------------------------------------------------------
Write-Host ""
Write-Host "=== ALL TESTS PASSED ===" -ForegroundColor Green
Write-Host ""
Write-Host "Final state:"
$r = Invoke-Api -Url "$Base/accounts" -Token $Token
$r.Body | ConvertTo-Json -Depth 5