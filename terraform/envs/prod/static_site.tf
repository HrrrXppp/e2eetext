# #28: the built client is served from a private S3 bucket through
# CloudFront, which also fronts the ALB for the API, WebSocket, /health,
# /sitemap.xml and /robots.txt paths. Staged cutover: this adds the
# distribution next to the existing ALB client rules and EC2 client
# container. DNS keeps pointing at the ALB until the owner switches it to
# output static_site_domain_name. A follow-up change then removes the client
# target group, rules and container.

# CloudFront only accepts ACM certificates from us-east-1.
provider "aws" {
  alias  = "us_east_1"
  region = "us-east-1"
}

locals {
  # Everything the server owns, from the same inputs as the ALB rules.
  static_site_alb_path_patterns = concat(
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

  certificate_arn   = var.static_site_certificate_arn
  alb_dns_name      = module.alb.alb_dns_name
  alb_path_patterns = local.static_site_alb_path_patterns

  tags = var.tags
}
