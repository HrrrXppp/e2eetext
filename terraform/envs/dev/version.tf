# Deployed version for dig. One string is the e2eetext-server image tag and
# the CloudFront client directory. Prod has its own file and may differ.
# Edit this value and terraform apply.

locals {
  deployed_version = "v0.5.2"
}
