output "acm_certificate_arn" {
  description = "ACM certificate ARN on the prod HTTPS listener."
  value       = local.prod_acm_certificate_arn
}

output "acm_dns_validation" {
  description = "CNAME records to create so the Terraform-managed prod certificate can be issued. Empty when acm_certificate_arn is set."
  value = try([
    for dvo in aws_acm_certificate.prod[0].domain_validation_options : {
      domain = dvo.domain_name
      name   = dvo.resource_record_name
      type   = dvo.resource_record_type
      value  = dvo.resource_record_value
    }
  ], [])
}

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

output "static_site_bucket_name" {
  description = "S3 bucket for the prod client build (STATIC_SITE_BUCKET_PROD for .github/workflows/deploy-client.yml)."
  value       = module.static_site.bucket_name
}

output "static_site_distribution_id" {
  description = "prod CloudFront distribution ID (STATIC_SITE_DISTRIBUTION_ID_PROD for .github/workflows/deploy-client.yml)."
  value       = module.static_site.distribution_id
}

output "static_site_domain_name" {
  description = "prod distribution *.cloudfront.net name, the DNS target for the public hostnames at cutover."
  value       = module.static_site.distribution_domain_name
}

output "static_site_certificate_dns_validation" {
  description = "CNAME records to create so the prod us-east-1 certificate can be issued. Empty when static_site_certificate_arn is set."
  value       = module.static_site.certificate_dns_validation
}
