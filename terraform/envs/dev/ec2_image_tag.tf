# Git-pinned ECR IMAGE_TAG for the dig EC2 boot-deploy unit.
# Edit this value and commit when dig should pull a different tag on boot.
# Prod has its own pin: terraform/envs/prod/ec2_image_tag.tf

locals {
  ec2_boot_deploy_image_tag = "v0.2.0"
}
