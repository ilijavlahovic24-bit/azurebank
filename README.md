AzureBank: Cloud-Native Retail Banking API
## Status: Active Development

## Core Philosophy
A production-grade banking REST API built to demonstrate cloud-native development practices -
infrastructure as code, automated CI/CD pipelines, and secrets management on Microsoft Azure.
The focus is on deployment correctness and operational reliability rather than complex financial domain logic.

## What AzureBank Does
AzureBank provides a retail banking backend with:
- Account management (checking and savings accounts)
- Deposits, withdrawals, and account-to-account transfers (ACID guaranteed)
- JWT authentication with secrets managed via Azure Key Vault
- Fully automated CI/CD pipeline from commit to live deployment

## Project Structure

AzureBank/
├── cmd/api/        # Entrypoint
├── internal/
│   ├── auth/           # JWT auth
│   ├── accounts/       # Account domain
│   └── transactions/   # Deposit, withdraw, transfer
├── db/migrations/      # SQL migrations (golang-migrate)
├── terraform/          # Azure infrastructure as code
├── .github/workflows/  # CI (test + lint) and CD (build + deploy)
├── Dockerfile
└── docker-compose.yml  # Local dev with Postgres

## Azure Infrastructure
- App Service        - runs the containerized Go API
- Container Registry - stores Docker images
- PostgreSQL         - flexible server
- Key Vault          - JWT secret, database credentials

### In Progress
- Terraform provisioning (App Service, ACR, PostgreSQL, Key Vault)
- GitHub Actions CI/CD pipeline
- Core API endpoints (accounts, transactions)

### Planned
- Managed Identity integration for Key Vault access
- Health check endpoint + App Insights logging
- golang-migrate integration

## References and Documentation
