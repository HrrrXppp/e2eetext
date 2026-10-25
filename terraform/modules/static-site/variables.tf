variable "name" {
  description = "Base name for the CloudFront resources (e.g. \"e2eetext-dev\"). Cache and response-header policy names must be unique per account, so this must differ between environments."
  type        = string
}

variable "bucket_name" {
  description = "Name of the private S3 bucket that holds the built client (client/dist). Bucket names are global, so pick one that is unlikely to collide. Must match static_site_bucket_names in envs/shared so the GitHub Actions role may upload to it."
  type        = string
}

variable "aliases" {
  description = "Public hostnames served by the distribution. The first one is the primary name on the certificate, the rest are subject alternative names."
  type        = list(string)

  validation {
    condition     = length(var.aliases) > 0
    error_message = "aliases must contain at least one hostname."
  }
}

variable "certificate_arn" {
  description = <<-EOT
    Existing ACM certificate ARN in us-east-1 (CloudFront only accepts
    certificates from that region). Leave empty to create a DNS-validated
    certificate for aliases. Apply then waits until it is ISSUED: add the
    CNAMEs from output certificate_dns_validation at the DNS host (this
    account has no Route 53 hosted zones).
  EOT
  type        = string
  default     = ""
}

variable "alb_dns_name" {
  description = "DNS name of the Application Load Balancer. CloudFront forwards the API, WebSocket, /health, /sitemap.xml and /robots.txt paths to it over HTTPS."
  type        = string
}

variable "alb_path_patterns" {
  description = "CloudFront path patterns forwarded to the ALB with caching disabled (for example /api/*, /health, /sitemap.xml, /robots.txt). Every other path is served from S3."
  type        = list(string)
}

variable "client_version" {
  description = "Directory in the bucket that CloudFront serves the client from (the image-style tag, for example \"v0.4.1\"), set as the S3 origin path. deploy-client.yml uploads each release to its own directory; change this to the new tag and apply to switch releases. Empty serves the bucket root."
  type        = string
  default     = ""
}

variable "price_class" {
  description = "CloudFront price class."
  type        = string
  default     = "PriceClass_100"
}

variable "tags" {
  description = "Tags applied to the bucket, distribution and certificate."
  type        = map(string)
  default     = {}
}
