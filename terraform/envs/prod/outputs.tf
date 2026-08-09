output "alb_dns_name" {
  description = "DNS name of the prod ALB — point your production domain's Route 53 alias/CNAME here."
  value       = module.alb.alb_dns_name
}

output "instance_id" {
  description = "Prod EC2 instance ID."
  value       = module.ec2.instance_id
}

output "rds_endpoint" {
  description = "Prod RDS connection endpoint (\"address:port\") — build DATABASE_URL from this plus rds_username/rds_app_database_name and the real password."
  value       = module.rds.endpoint
}

output "rds_app_database_name" {
  description = "PostgreSQL database name for DATABASE_URL on the prod instance."
  value       = module.rds.app_database_name
}

output "rds_cron_database_name" {
  description = "Effective cron.database_name for prod RDS pg_cron (defaults to rds_app_database_name)."
  value       = module.rds.cron_database_name
}
