locals {
  # try() so a count=0 certificate (explicit acm_certificate_arn) does not
  # make the unused branch fail planning.
  prod_acm_certificate_arn = coalesce(
    var.acm_certificate_arn != "" ? var.acm_certificate_arn : null,
    try(aws_acm_certificate_validation.prod[0].certificate_arn, null),
  )
}

# Prod HTTPS certificate in the ALB's region. A us-east-1 certificate cannot
# be attached to a us-east-2 listener, and the issued us-east-2 certificate
# used by dig covers only the dig hostnames.
#
# This account has no Route 53 hosted zones, so Terraform cannot publish the
# validation CNAMEs. aws_acm_certificate_validation waits until the
# certificate is ISSUED after those records are created at the DNS host
# (output acm_dns_validation; also visible on the pending certificate in
# the ACM console while apply is waiting).

resource "aws_acm_certificate" "prod" {
  count = var.acm_certificate_arn == "" ? 1 : 0

  domain_name               = var.acm_domain_name
  subject_alternative_names = var.acm_subject_alternative_names
  validation_method         = "DNS"

  lifecycle {
    create_before_destroy = true
  }

  tags = var.tags
}

resource "aws_acm_certificate_validation" "prod" {
  count = var.acm_certificate_arn == "" ? 1 : 0

  certificate_arn = aws_acm_certificate.prod[0].arn

  timeouts {
    create = "45m"
  }
}
