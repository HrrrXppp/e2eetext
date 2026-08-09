output "endpoint" {
  description = "Connection endpoint (\"address:port\") — build DATABASE_URL from this plus var.username/app_database_name and the real password."
  value       = aws_db_instance.this.endpoint
}

output "address" {
  description = "Hostname of the instance, without the port."
  value       = aws_db_instance.this.address
}

output "port" {
  description = "Port the instance listens on."
  value       = aws_db_instance.this.port
}

output "arn" {
  description = "ARN of the DB instance."
  value       = aws_db_instance.this.arn
}

output "db_name" {
  description = "CreateDBInstance DBName attribute (often null on brownfield imports). Prefer app_database_name for DATABASE_URL."
  value       = aws_db_instance.this.db_name
}

output "app_database_name" {
  description = "PostgreSQL database name for DATABASE_URL / cron.database_name (may differ from CreateDBInstance DBName)."
  value       = local.app_database_name_effective
}

output "cron_database_name" {
  description = "Effective cron.database_name written to the parameter group when enable_pg_cron is true (app DB by default)."
  value       = local.cron_database_name
}

output "security_group_id" {
  description = "Security group ID created for the instance, or null when pre-existing security_group_ids were passed."
  value       = local.create_security_group ? aws_security_group.this[0].id : null
}

output "subnet_group_name" {
  description = "Name of the DB subnet group in use (created by this module, or the pre-existing one referenced by var.subnet_group_name when create_subnet_group is false)."
  value       = local.subnet_group_name_effective
}

output "parameter_group_name" {
  description = "Name of the DB parameter group in use (created by this module, or the pre-existing one referenced by var.parameter_group_name)."
  value       = local.parameter_group_name_effective
}
