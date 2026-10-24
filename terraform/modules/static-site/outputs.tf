output "bucket_name" {
  description = "S3 bucket the deploy workflow syncs client/dist into."
  value       = aws_s3_bucket.this.bucket
}

output "distribution_id" {
  description = "CloudFront distribution ID (the deploy workflow invalidates /index.html on it)."
  value       = aws_cloudfront_distribution.this.id
}

output "distribution_domain_name" {
  description = "The distribution's *.cloudfront.net name. Point the public hostnames' DNS (CNAME / ALIAS) here at cutover."
  value       = aws_cloudfront_distribution.this.domain_name
}

output "certificate_arn" {
  description = "us-east-1 ACM certificate ARN attached to the distribution."
  value       = local.certificate_arn
}

output "certificate_dns_validation" {
  description = "CNAME records to create so the Terraform-managed certificate can be issued. Empty when certificate_arn is set."
  value = try([
    for dvo in aws_acm_certificate.this[0].domain_validation_options : {
      domain = dvo.domain_name
      name   = dvo.resource_record_name
      type   = dvo.resource_record_type
      value  = dvo.resource_record_value
    }
  ], [])
}
