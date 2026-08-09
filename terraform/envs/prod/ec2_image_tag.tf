# Git-pinned ECR IMAGE_TAG for the prod EC2 boot-deploy unit.
# Edit this value and commit when prod should pull a different tag on boot.
# Dig has its own pin: terraform/envs/dev/ec2_image_tag.tf

locals {
  ec2_boot_deploy_image_tag = "v0.2.0"
}
