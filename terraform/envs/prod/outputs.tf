output "alb_dns_name" {
  description = "DNS name of the prod ALB — point your production domain's Route 53 alias/CNAME here."
  value       = module.alb.alb_dns_name
}

output "instance_id" {
  description = "Prod EC2 instance ID."
  value       = module.ec2.instance_id
}

output "rds_endpoint" {
  description = "Prod RDS connection endpoint (\"address:port\") — build DATABASE_URL from this plus rds_username/rds_db_name and the real password."
  value       = module.rds.endpoint
}
