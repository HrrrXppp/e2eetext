# Image tags the prod EC2 boot-deploy association pulls.
# Change a default and run terraform apply in this directory.
# Apply updates the association, which logs into ECR and recreates the
# container for that tag. Dig has its own pins: terraform/envs/dev/ec2_image_tag.tf
#
# ec2_boot_deploy_image_tag is the server container (e2eetext-server).
# ec2_boot_deploy_client_image_tag is the client container still running
# on the instance (e2eetext-client). New releases do not publish that
# image; v0.4.1 is the newest tag in the registry. The CloudFront client
# is static_site_client_version, not this tag.

variable "ec2_boot_deploy_image_tag" {
  description = "ECR tag for the prod e2eetext-server container. Apply pulls this tag onto the instance."
  type        = string
  default     = "v0.5.1"
}

variable "ec2_boot_deploy_client_image_tag" {
  description = "ECR tag for the prod e2eetext-client container. Apply pulls this tag. Empty leaves the running client container unchanged. New releases do not publish this image; v0.4.1 is the newest tag."
  type        = string
  default     = "v0.4.1"
}
