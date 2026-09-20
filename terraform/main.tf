terraform {
  required_version = ">= 1.5"

  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 4.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
  }
}

provider "azurerm" {
  features {
    resource_group {
      prevent_deletion_if_contains_resources = false
    }
  }
}

# --- Random suffix za globalno-unikatna imena ---
resource "random_string" "suffix" {
  length  = 6
  special = false
  upper   = false
}

locals {
  tags = {
    Project = "azurebank"
    Managed = "terraform"
  }
  acr_name = "${var.prefix}acr${random_string.suffix.result}"
  app_name = "${var.prefix}-app-${random_string.suffix.result}"
  pg_name  = "${var.prefix}-pg-${random_string.suffix.result}"
}

# --- Resource Group ---
resource "azurerm_resource_group" "main" {
  name     = "${var.prefix}-rg"
  location = var.location
  tags     = local.tags
}

# --- PostgreSQL Flexible Server ---
resource "azurerm_postgresql_flexible_server" "main" {
  name                   = local.pg_name
  resource_group_name    = azurerm_resource_group.main.name
  location               = azurerm_resource_group.main.location
  version                = "15"
  administrator_login    = var.db_admin_login
  administrator_password = var.db_admin_password

  storage_mb = 32768
  sku_name   = "B_Standard_B1ms"
  zone       = "1"

  backup_retention_days        = 7
  geo_redundant_backup_enabled = false

  public_network_access_enabled = true

  tags = local.tags
}

resource "azurerm_postgresql_flexible_server_database" "main" {
  name      = "azurebank"
  server_id = azurerm_postgresql_flexible_server.main.id
  charset   = "UTF8"
  collation = "en_US.utf8"
}

resource "azurerm_postgresql_flexible_server_firewall_rule" "azure" {
  name             = "AllowAzureServices"
  server_id        = azurerm_postgresql_flexible_server.main.id
  start_ip_address = "0.0.0.0"
  end_ip_address   = "0.0.0.0"
}

# --- Container Registry ---
resource "azurerm_container_registry" "main" {
  name                = local.acr_name
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
  sku                 = "Basic"
  admin_enabled       = true

  tags = local.tags
}

# --- App Service Plan ---
resource "azurerm_service_plan" "main" {
  name                = "${var.prefix}-plan"
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
  os_type             = "Linux"
  sku_name            = "B1"

  tags = local.tags
}

# --- Linux Web App (containers) ---
resource "azurerm_linux_web_app" "main" {
  name                = local.app_name
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_service_plan.main.location
  service_plan_id     = azurerm_service_plan.main.id

  site_config {
    always_on = false

    application_stack {
      docker_image_name        = "azurebank:latest"
      docker_registry_url      = "https://${azurerm_container_registry.main.login_server}"
      docker_registry_username = azurerm_container_registry.main.admin_username
      docker_registry_password = azurerm_container_registry.main.admin_password
    }
  }

  app_settings = {
    "PORT"          = "8080"
    "WEBSITES_PORT" = "8080"
    "JWT_SECRET"    = var.jwt_secret
    "DATABASE_URL"  = "postgres://${var.db_admin_login}:${var.db_admin_password}@${azurerm_postgresql_flexible_server.main.fqdn}:5432/azurebank?sslmode=require"
  }

  tags = local.tags
}