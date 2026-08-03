output "alb_dns_name" {
  description = "DNS name of the dev ALB — point your dev domain's Route 53 alias/CNAME here."
  value       = module.alb.alb_dns_name
}

output "instance_id" {
  description = "Dev EC2 instance ID."
  value       = module.ec2.instance_id
}

output "rds_endpoint" {
  description = "Dev RDS connection endpoint (\"address:port\") — build DATABASE_URL from this plus rds_username/rds_app_database_name and the real password."
  value       = module.rds.endpoint
}

output "rds_app_database_name" {
  description = "PostgreSQL database name for DATABASE_URL on the live-dev instance (dev_e2eetext)."
  value       = module.rds.app_database_name
}
