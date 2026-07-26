# Phase 2 of #27: dev's networking/ALB/EC2. A live dev EC2/ALB/(VPC) stack
# already exists (confirmed by the repo owner on #27, 2026-07-26) — this
# root config describes those existing resources for `terraform import`,
# the same pattern Phase 1 used for the manually-created ECR repos and
# GitHub OIDC role. See README.md's "Infrastructure as code (Terraform)"
# section for the required one-time `terraform import` bootstrap before the
# first `terraform apply` here. Do NOT `terraform apply` before importing —
# it would try to create resources that already exist and either fail
# outright or, worse, succeed in creating duplicates alongside the live
# ones.

provider "aws" {
  region = var.aws_region
}

module "network" {
  source = "../../modules/network"

  vpc_id     = var.vpc_id
  subnet_ids = var.subnet_ids
}

module "ec2" {
  source = "../../modules/ec2"

  name          = var.name_prefix
  ami_id        = var.ami_id
  instance_type = var.instance_type
  subnet_id     = var.ec2_subnet_id != "" ? var.ec2_subnet_id : module.network.subnet_ids[0]
  vpc_id        = module.network.vpc_id

  alb_security_group_id = var.alb_security_group_id
  admin_ssh_cidr_blocks = var.admin_ssh_cidr_blocks
  key_name              = var.key_name

  security_group_name       = var.ec2_security_group_name
  iam_role_name             = var.ec2_iam_role_name
  iam_instance_profile_name = var.ec2_iam_instance_profile_name

  tags = var.tags
}

module "alb" {
  source = "../../modules/alb"

  name       = var.name_prefix
  vpc_id     = module.network.vpc_id
  subnet_ids = length(var.alb_subnet_ids) > 0 ? var.alb_subnet_ids : module.network.subnet_ids

  security_group_ids  = [var.alb_security_group_id]
  instance_id         = module.ec2.instance_id
  acm_certificate_arn = var.acm_certificate_arn

  tags = var.tags
}
