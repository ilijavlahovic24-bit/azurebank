output "resource_group" {
  value = azurerm_resource_group.main.name
}

output "acr_login_server" {
  value = azurerm_container_registry.main.login_server
}

output "acr_username" {
  value = azurerm_container_registry.main.admin_username
}

output "acr_password" {
  value     = azurerm_container_registry.main.admin_password
  sensitive = true
}

output "app_url" {
  value = "https://${azurerm_linux_web_app.main.default_hostname}"
}

output "postgres_fqdn" {
  value = azurerm_postgresql_flexible_server.main.fqdn
}

output "db_url" {
  value     = "postgres://${var.db_admin_login}:${var.db_admin_password}@${azurerm_postgresql_flexible_server.main.fqdn}:5432/azurebank?sslmode=require"
  sensitive = true
}