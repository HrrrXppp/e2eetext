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

output "rds_cron_database_name" {
  description = "Effective cron.database_name for dig RDS pg_cron (defaults to rds_app_database_name)."
  value       = module.rds.cron_database_name
}

output "static_site_bucket_name" {
  description = "S3 bucket for the dev client build (STATIC_SITE_BUCKET_DEV for .github/workflows/deploy-client.yml)."
  value       = module.static_site.bucket_name
}

output "static_site_distribution_id" {
  description = "dev CloudFront distribution ID (STATIC_SITE_DISTRIBUTION_ID_DEV for .github/workflows/deploy-client.yml)."
  value       = module.static_site.distribution_id
}

output "static_site_domain_name" {
  description = "dev distribution *.cloudfront.net name, the DNS target for the public hostnames at cutover."
  value       = module.static_site.distribution_domain_name
}

output "static_site_certificate_dns_validation" {
  description = "CNAME records to create so the dev us-east-1 certificate can be issued. Empty when static_site_certificate_arn is set."
  value       = module.static_site.certificate_dns_validation
}
