# Static client hosting for #28: the built SPA (client/dist) lives in a
# private S3 bucket and CloudFront is the single public entry point. The
# API, WebSocket, /health, /sitemap.xml and /robots.txt paths are forwarded
# to the ALB (caching off, every header, cookie and query string forwarded)
# so the browser keeps talking to one host: no CORS or cookie changes.

locals {
  s3_origin_id  = "s3-client"
  alb_origin_id = "alb-server"

  certificate_arn = var.certificate_arn != "" ? var.certificate_arn : aws_acm_certificate_validation.this[0].certificate_arn

  # AWS managed policies (stable IDs).
  cache_policy_disabled     = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad" # Managed-CachingDisabled
  origin_request_policy_all = "216adef6-5c7f-47e4-b989-5492eafa07d3" # Managed-AllViewer (forwards Host)

  alb_methods = ["GET", "HEAD", "OPTIONS", "PUT", "POST", "PATCH", "DELETE"]
}

# --- Certificate (CloudFront requires us-east-1) ---

resource "aws_acm_certificate" "this" {
  provider = aws.us_east_1
  count    = var.certificate_arn == "" ? 1 : 0

  domain_name               = var.aliases[0]
  subject_alternative_names = slice(var.aliases, 1, length(var.aliases))
  validation_method         = "DNS"

  lifecycle {
    create_before_destroy = true
  }

  tags = var.tags
}

# Waits until the certificate is ISSUED, i.e. until the validation CNAMEs
# (output certificate_dns_validation) exist at the DNS host.
resource "aws_acm_certificate_validation" "this" {
  provider = aws.us_east_1
  count    = var.certificate_arn == "" ? 1 : 0

  certificate_arn = aws_acm_certificate.this[0].arn

  timeouts {
    create = "45m"
  }
}

# --- Bucket (private; only this distribution may read it) ---

resource "aws_s3_bucket" "this" {
  bucket = var.bucket_name

  tags = var.tags
}

resource "aws_s3_bucket_public_access_block" "this" {
  bucket = aws_s3_bucket.this.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_ownership_controls" "this" {
  bucket = aws_s3_bucket.this.id

  rule {
    object_ownership = "BucketOwnerEnforced"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "this" {
  bucket = aws_s3_bucket.this.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

data "aws_iam_policy_document" "bucket" {
  statement {
    sid       = "AllowCloudFrontRead"
    effect    = "Allow"
    actions   = ["s3:GetObject"]
    resources = ["${aws_s3_bucket.this.arn}/*"]

    principals {
      type        = "Service"
      identifiers = ["cloudfront.amazonaws.com"]
    }

    condition {
      test     = "StringEquals"
      variable = "AWS:SourceArn"
      values   = [aws_cloudfront_distribution.this.arn]
    }
  }
}

resource "aws_s3_bucket_policy" "this" {
  bucket = aws_s3_bucket.this.id
  policy = data.aws_iam_policy_document.bucket.json

  depends_on = [aws_s3_bucket_public_access_block.this]
}

# --- CloudFront ---

resource "aws_cloudfront_origin_access_control" "this" {
  name                              = "${var.name}-static-site"
  description                       = "Read access to the ${var.bucket_name} bucket"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

# SPA fallback: any extension-less path (/chats, /roadmap, /oauth/callback)
# is rewritten to /index.html. Paths with a file extension are left alone,
# so a missing asset stays an error instead of becoming HTML. The ALB
# behaviors do not run this function, so unknown API paths still return the
# server's own 404.
resource "aws_cloudfront_function" "spa_fallback" {
  name    = "${var.name}-spa-fallback"
  runtime = "cloudfront-js-2.0"
  comment = "Rewrite extension-less paths to /index.html"
  publish = true

  code = <<-EOT
    function handler(event) {
      var request = event.request;
      var uri = request.uri;
      var last = uri.substring(uri.lastIndexOf('/') + 1);
      if (uri === '/' || last.indexOf('.') === -1) {
        request.uri = '/index.html';
      }
      return request;
    }
  EOT
}

# Honors the object's own Cache-Control: hashed /assets/* files are uploaded
# with a one-year immutable max-age, index.html and instance.json with
# no-cache. With no header the TTL is 0, so a forgotten header fails safe.
resource "aws_cloudfront_cache_policy" "static" {
  name        = "${var.name}-static-site"
  comment     = "Honor origin Cache-Control for the S3 client files"
  min_ttl     = 0
  default_ttl = 0
  max_ttl     = 31536000

  parameters_in_cache_key_and_forwarded_to_origin {
    enable_accept_encoding_gzip   = true
    enable_accept_encoding_brotli = true

    cookies_config {
      cookie_behavior = "none"
    }

    headers_config {
      header_behavior = "none"
    }

    query_strings_config {
      query_string_behavior = "none"
    }
  }
}

resource "aws_cloudfront_response_headers_policy" "static" {
  name    = "${var.name}-static-site"
  comment = "Security headers for the S3 client files"

  security_headers_config {
    content_type_options {
      override = true
    }

    referrer_policy {
      referrer_policy = "strict-origin-when-cross-origin"
      override        = false
    }

    strict_transport_security {
      access_control_max_age_sec = 31536000
      include_subdomains         = false
      preload                    = false
      override                   = false
    }
  }
}

resource "aws_cloudfront_distribution" "this" {
  enabled             = true
  is_ipv6_enabled     = true
  http_version        = "http2and3"
  comment             = "${var.name} client (S3) and server (ALB)"
  aliases             = var.aliases
  default_root_object = "index.html"
  price_class         = var.price_class

  origin {
    origin_id                = local.s3_origin_id
    domain_name              = aws_s3_bucket.this.bucket_regional_domain_name
    origin_path              = var.client_version == "" ? null : "/${var.client_version}"
    origin_access_control_id = aws_cloudfront_origin_access_control.this.id
  }

  origin {
    origin_id   = local.alb_origin_id
    domain_name = var.alb_dns_name

    custom_origin_config {
      http_port                = 80
      https_port               = 443
      origin_protocol_policy   = "https-only"
      origin_ssl_protocols     = ["TLSv1.2"]
      origin_read_timeout      = 60
      origin_keepalive_timeout = 60
    }
  }

  default_cache_behavior {
    target_origin_id           = local.s3_origin_id
    viewer_protocol_policy     = "redirect-to-https"
    allowed_methods            = ["GET", "HEAD", "OPTIONS"]
    cached_methods             = ["GET", "HEAD"]
    compress                   = true
    cache_policy_id            = aws_cloudfront_cache_policy.static.id
    response_headers_policy_id = aws_cloudfront_response_headers_policy.static.id

    function_association {
      event_type   = "viewer-request"
      function_arn = aws_cloudfront_function.spa_fallback.arn
    }
  }

  # Vite's hashed bundle files: no SPA rewrite, so a missing asset is a
  # plain error.
  ordered_cache_behavior {
    path_pattern               = "/assets/*"
    target_origin_id           = local.s3_origin_id
    viewer_protocol_policy     = "redirect-to-https"
    allowed_methods            = ["GET", "HEAD", "OPTIONS"]
    cached_methods             = ["GET", "HEAD"]
    compress                   = true
    cache_policy_id            = aws_cloudfront_cache_policy.static.id
    response_headers_policy_id = aws_cloudfront_response_headers_policy.static.id
  }

  # API, WebSocket, /health, /sitemap.xml, /robots.txt -> ALB. Caching is
  # off and the viewer's Host header is forwarded, so the server sees the
  # public hostname (the sitemap is generated per host) and the ALB
  # certificate is validated against it.
  dynamic "ordered_cache_behavior" {
    for_each = var.alb_path_patterns

    content {
      path_pattern             = ordered_cache_behavior.value
      target_origin_id         = local.alb_origin_id
      viewer_protocol_policy   = "redirect-to-https"
      allowed_methods          = local.alb_methods
      cached_methods           = ["GET", "HEAD"]
      compress                 = false
      cache_policy_id          = local.cache_policy_disabled
      origin_request_policy_id = local.origin_request_policy_all
    }
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    acm_certificate_arn      = local.certificate_arn
    ssl_support_method       = "sni-only"
    minimum_protocol_version = "TLSv1.2_2021"
  }

  tags = var.tags
}
