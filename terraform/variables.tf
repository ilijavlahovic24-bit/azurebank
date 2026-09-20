variable "location" {
  description = "Azure region"
  type        = string
  default     = "northeurope"
}

variable "prefix" {
  description = "Prefix for all resource names"
  type        = string
  default     = "azurebank"
}

variable "db_admin_login" {
  description = "PostgreSQL admin username"
  type        = string
  default     = "azurebankadmin"
}

variable "db_admin_password" {
  description = "PostgreSQL admin password"
  type        = string
  sensitive   = true
}

variable "jwt_secret" {
  description = "JWT signing secret"
  type        = string
  sensitive   = true
}