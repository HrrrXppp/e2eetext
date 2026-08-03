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
  additional_https_rules   = var.additional_https_rules

  tags = var.tags
}

# --- RDS (Phase 3 of #27) ---
# Same "live status not yet confirmed" treatment as the ec2/alb modules
# above (see this file's header) — Phase 3 was additionally implemented
# with no AWS credentials at all, so unlike dev, there isn't even
# circumstantial evidence (like dev's "ec2-rds-1" security group) here.
# Confirm whether a live prod RDS instance exists before planning/applying:
#
#   aws rds describe-db-instances --filters "Name=db-instance-id,Values=e2eetext-prod*"
#
# - If real resources come back: audit them fully (engine version, class,
#   storage, subnet/parameter groups, ...) and follow the same import
#   bootstrap as dev (README's Terraform section) with prod's real
#   identifiers substituted in.
# - If nothing comes back: `terraform apply` can create the instance fresh
#   — but get sign-off on instance class/storage/Multi-AZ choices first,
#   same as any other prod infrastructure decision.
module "rds" {
  source = "../../modules/rds"

  identifier     = var.rds_identifier
  engine_version = var.rds_engine_version
  instance_class = var.rds_instance_class

  allocated_storage = var.rds_allocated_storage
  storage_type      = var.rds_storage_type
  storage_encrypted = var.rds_storage_encrypted

  db_name  = var.rds_db_name
  username = var.rds_username
  password = var.rds_password

  vpc_id              = module.network.vpc_id
  subnet_ids          = var.rds_subnet_ids
  subnet_group_name   = var.rds_subnet_group_name
  create_subnet_group = var.rds_create_subnet_group
  security_group_ids  = var.rds_security_group_ids

  security_group_name = var.rds_security_group_name
  # See envs/dev/main.tf for why this isn't coalesce(...) — coalesce()
  # errors when ALL arguments are null/empty-string, which is exactly what
  # happens whenever module.ec2.security_group_id is null (PR #32 comment:
  # real "Fix error in terraform plan dev" failure once live tfvars were
  # configured; prod's own module.ec2 currently creates its SG so this
  # branch hasn't fired live here, but the expression is identical and just
  # as capable of erroring).
  ingress_security_group_id = var.rds_ingress_security_group_id != "" ? var.rds_ingress_security_group_id : (module.ec2.security_group_id != null ? module.ec2.security_group_id : "")

  create_parameter_group   = var.rds_create_parameter_group
  parameter_group_name     = var.rds_parameter_group_name
  parameter_group_family   = var.rds_parameter_group_family
  enable_pg_cron           = var.rds_enable_pg_cron
  cron_database_name       = var.rds_cron_database_name
  shared_preload_libraries = var.rds_shared_preload_libraries

  multi_az                = var.rds_multi_az
  publicly_accessible     = var.rds_publicly_accessible
  backup_retention_period = var.rds_backup_retention_period
  backup_window           = var.rds_backup_window
  maintenance_window      = var.rds_maintenance_window
  deletion_protection     = var.rds_deletion_protection
  skip_final_snapshot     = var.rds_skip_final_snapshot
  copy_tags_to_snapshot   = var.rds_copy_tags_to_snapshot
  network_type            = var.rds_network_type

  iops                  = var.rds_iops
  storage_throughput    = var.rds_storage_throughput
  kms_key_id            = var.rds_kms_key_id
  max_allocated_storage = var.rds_max_allocated_storage
  ca_cert_identifier    = var.rds_ca_cert_identifier

  monitoring_interval                   = var.rds_monitoring_interval
  monitoring_role_arn                   = var.rds_monitoring_role_arn
  performance_insights_enabled          = var.rds_performance_insights_enabled
  performance_insights_kms_key_id       = var.rds_performance_insights_kms_key_id
  performance_insights_retention_period = var.rds_performance_insights_retention_period
  enabled_cloudwatch_logs_exports       = var.rds_enabled_cloudwatch_logs_exports

  tags = var.tags
}
