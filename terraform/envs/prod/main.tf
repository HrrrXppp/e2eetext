# Phase 2 of #27: prod's networking/ALB/EC2.
#
# UNLIKE dev, whether a live prod EC2/ALB/(VPC) stack already exists has
# NOT been confirmed by the repo owner as part of this issue. The README's
# "Production on AWS (single EC2 + ALB)" walkthrough only documents a
# procedure — every value in it (account ID, region, domain, VPC/subnet/SG
# IDs, ACM cert ARN) is an obvious placeholder, not a recorded real
# identifier, so the repo alone doesn't prove a prod stack has actually been
# stood up by following it.
#
# This root config is written the same way as dev (import-shaped, not
# create-fresh) because that is the safer assumption if a live prod stack
# does turn out to exist — but before running EITHER `terraform import` OR
# `terraform apply` here, first confirm which situation is real, e.g.:
#
#   aws ec2 describe-instances --filters "Name=tag:Name,Values=e2eetext-prod*"
#   aws elbv2 describe-load-balancers --names e2eetext-prod
#
# - If real resources come back: follow the same import bootstrap as dev
#   (see README.md's "Infrastructure as code (Terraform)" section) with
#   prod's real identifiers substituted in.
# - If nothing comes back: this can be applied fresh instead, same as any
#   greenfield Terraform — but get that confirmed (and get sign-off on
#   instance size/domain/ACM cert choices) before running `apply` against
#   the resources real users would hit.
#
# Heads-up from the dev audit (2026-07-26): the live dev stack diverged
# from create-alb.example.sh in almost every override this config exposes
# (ALB/TG names, idle timeout, default action, rule priorities/patterns,
# instance SGs, no instance profile). If a live prod stack exists, expect
# the same and audit it before importing — the variables below default to
# the script-shaped topology and must be overridden to match reality.

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

  security_group_ids          = var.ec2_security_group_ids
  create_iam_instance_profile = var.create_ec2_iam_instance_profile

  security_group_name       = var.ec2_security_group_name
  iam_role_name             = var.ec2_iam_role_name
  iam_instance_profile_name = var.ec2_iam_instance_profile_name

  tags = var.tags
}

module "alb" {
  source = "../../modules/alb"

  name       = var.alb_name != "" ? var.alb_name : var.name_prefix
  vpc_id     = module.network.vpc_id
  subnet_ids = length(var.alb_subnet_ids) > 0 ? var.alb_subnet_ids : module.network.subnet_ids

  security_group_ids  = [var.alb_security_group_id]
  instance_id         = module.ec2.instance_id
  acm_certificate_arn = var.acm_certificate_arn

  server_target_group_name = var.server_target_group_name
  client_target_group_name = var.client_target_group_name
  idle_timeout             = var.alb_idle_timeout
  https_default_action     = var.https_default_action
  api_rule_priority        = var.api_rule_priority
  api_path_patterns        = var.api_path_patterns
  create_health_rule       = var.create_health_rule

  tags = var.tags
}
