# The built client is served from a private S3 bucket through CloudFront.
# The same distribution forwards the API, WebSocket, /health, /sitemap.xml
# and /robots.txt to the EC2 instance on port 8081.

# CloudFront only accepts ACM certificates from us-east-1.
provider "aws" {
  alias  = "us_east_1"
  region = "us-east-1"
}

locals {
  # Paths CloudFront forwards to the instance. Client paths stay on S3.
  static_site_server_path_patterns = concat(
    var.api_path_patterns,
    var.create_health_rule ? ["/health"] : [],
    flatten([for r in var.additional_https_rules : r.path_patterns if r.target == "server"]),
  )
}

module "static_site" {
  source = "../../modules/static-site"

  providers = {
    aws           = aws
    aws.us_east_1 = aws.us_east_1
  }

  name        = var.name_prefix
  bucket_name = var.static_site_bucket_name
  aliases     = var.static_site_aliases

  client_version       = local.deployed_version # pinned in version.tf
  certificate_arn      = var.static_site_certificate_arn
  server_origin_domain = module.ec2.public_dns
  server_path_patterns = local.static_site_server_path_patterns

  depends_on = [module.ec2]

  tags = var.tags
}
