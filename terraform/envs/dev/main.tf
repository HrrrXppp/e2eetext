# Phase 2 of #27: dev's networking/ALB/EC2. A live dev EC2/ALB/(VPC) stack
# already exists (confirmed by the repo owner on #27, 2026-07-26, and
# audited against AWS the same day) — this root config describes those
# existing resources for `terraform import`, the same pattern Phase 1 used
# for the manually-created ECR repos and GitHub OIDC role. See README.md's
# "Infrastructure as code (Terraform)" section for the required one-time
# `terraform import` bootstrap before the first `terraform apply` here. Do
# NOT `terraform apply` before importing — it would try to create resources
# that already exist and either fail outright or, worse, succeed in
# creating duplicates alongside the live ones.
#
# This root is plannable with no terraform.tfvars at all: every variable
# either has a confirmed-live default (audited 2026-07-26 — not guesses;
# see variables.tf) or is looked up from the live stack by name below, so
# nothing environment-specific (account ID, ARNs, sg-/subnet- IDs) needs to
# be committed to this public repo.

provider "aws" {
  region = var.aws_region
}

module "network" {
  source = "../../modules/network"

  vpc_id     = var.vpc_id
  subnet_ids = var.subnet_ids
}

# --- Live-stack lookups ---
# The dev ALB's security group and ACM certificate are pre-existing inputs,
# not resources this config manages. Rather than requiring their IDs/ARNs
# in a tfvars file (the account-specific ARN can't be committed here), look
# them up from the live ALB by its name. If the ALB doesn't exist (e.g.
# standing dev up fresh in a new account), these lookups fail — set
# var.alb_security_group_id / var.acm_certificate_arn explicitly instead.

data "aws_lb" "live" {
  count = var.alb_security_group_id == "" || var.acm_certificate_arn == "" ? 1 : 0

  name = var.alb_name
}

data "aws_lb_listener" "live_https" {
  count = var.acm_certificate_arn == "" ? 1 : 0

  load_balancer_arn = data.aws_lb.live[0].arn
  port              = 443
}

# The live dev instance runs with hand-made security groups (by name:
# "launch-wizard-2" and "ec2-rds-1" — audited 2026-07-26), not one this
# config creates. Resolve their names to IDs so the IDs themselves stay out
# of the repo.
data "aws_security_group" "ec2" {
  for_each = length(var.ec2_security_group_ids) == 0 ? toset(var.ec2_security_group_names) : toset([])

  name   = each.value
  vpc_id = module.network.vpc_id
}

locals {
  alb_security_group_id = var.alb_security_group_id != "" ? var.alb_security_group_id : tolist(data.aws_lb.live[0].security_groups)[0]
  acm_certificate_arn   = var.acm_certificate_arn != "" ? var.acm_certificate_arn : data.aws_lb_listener.live_https[0].certificate_arn

  ec2_security_group_ids = length(var.ec2_security_group_ids) > 0 ? var.ec2_security_group_ids : [
    for name in var.ec2_security_group_names : data.aws_security_group.ec2[name].id
  ]
}

module "ec2" {
  source = "../../modules/ec2"

  name          = var.name_prefix
  ami_id        = var.ami_id
  instance_type = var.instance_type
  subnet_id     = var.ec2_subnet_id != "" ? var.ec2_subnet_id : module.network.subnet_ids[0]
  vpc_id        = module.network.vpc_id

  alb_security_group_id = local.alb_security_group_id
  admin_ssh_cidr_blocks = var.admin_ssh_cidr_blocks
  key_name              = var.key_name

  # The live instance's SGs and (absent) instance profile — see
  # variables.tf; both defaults describe what's actually live so import
  # stays zero-diff.
  security_group_ids          = local.ec2_security_group_ids
  create_iam_instance_profile = var.create_ec2_iam_instance_profile

  security_group_name       = var.ec2_security_group_name
  iam_role_name             = var.ec2_iam_role_name
  iam_instance_profile_name = var.ec2_iam_instance_profile_name

  tags = var.tags
}

module "alb" {
  source = "../../modules/alb"

  name       = var.alb_name
  vpc_id     = module.network.vpc_id
  subnet_ids = length(var.alb_subnet_ids) > 0 ? var.alb_subnet_ids : module.network.subnet_ids

  security_group_ids  = [local.alb_security_group_id]
  instance_id         = module.ec2.instance_id
  acm_certificate_arn = local.acm_certificate_arn

  # The live dev ALB diverges from create-alb.example.sh (the module's
  # defaults) in all of the following, audited 2026-07-26 — these values
  # describe what's actually live:
  server_target_group_name = var.server_target_group_name # live: "dev-server"
  client_target_group_name = var.client_target_group_name # live: "dev-client"
  idle_timeout             = var.alb_idle_timeout         # live: 60, not 3600
  https_default_action     = var.https_default_action     # live: fixed 404, not forward-to-client
  api_rule_priority        = var.api_rule_priority        # live: 100, not 10
  api_path_patterns        = var.api_path_patterns        # live: ["/api/*"], not ["/api*"]
  create_health_rule       = var.create_health_rule       # live: no /health rule
  additional_https_rules   = var.additional_https_rules   # live: SPA rules 10–14

  health_check_healthy_threshold   = var.health_check_healthy_threshold   # live: 5, not 3
  health_check_unhealthy_threshold = var.health_check_unhealthy_threshold # live: 2, not 3
  server_health_check_port         = var.server_health_check_port         # live: "8081", not "traffic-port"
  client_health_check_port         = var.client_health_check_port         # live: "traffic-port"
  client_health_check_path         = var.client_health_check_path         # live: "/"

  tags = var.tags
}

# --- RDS (Phase 3 of #27) ---
# UNLIKE network/ec2/alb above, this section's variables have NO
# confirmed-live defaults. Phase 2's audit (2026-07-26) found the live dev
# EC2 instance attached to a security group named "ec2-rds-1" whose own
# description warns that detaching it can cut RDS connectivity — strong
# evidence a live dev RDS instance already exists — but this phase was
# implemented with no AWS credentials available, so none of that instance's
# actual engine version / instance class / storage / subnet-group /
# parameter-group details could be verified. Every identifying attribute
# below is therefore required with no default (rds_ingress_security_group_id
# excepted — see its variables.tf comment for its narrower fallback, now
# backstopped by modules/rds's own precondition — PR #32 review), the same
# treatment Phase 2 gave envs/prod while its live status was unconfirmed.
# rds_subnet_ids previously fell back to module.network's (public) subnets
# when unset; that silent fallback was removed (PR #32 review) since RDS is
# commonly private — it's now required with no default, same as the rest of
# this block. Fill in
# terraform.tfvars from a real `aws rds describe-db-instances` /
# `describe-db-parameter-groups` audit before planning or importing — see
# README's Terraform section.
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
  # module.ec2's own SG is null here (dev's EC2 instance uses pre-existing,
  # hand-made SGs — see ec2_security_group_names above), so this falls back
  # to "" unless var.rds_ingress_security_group_id is set explicitly — do
  # so before import (likely "ec2-rds-1"'s ID). If rds_security_group_ids
  # is also left empty (module creates its own SG), modules/rds's own
  # precondition now fails the plan on an empty ingress SG here, rather
  # than silently creating an SG with zero ingress rules (PR #32 review).
  # coalesce(module.ec2.security_group_id, "") used to error here: when
  # module.ec2.security_group_id is null (dev's real case), coalesce()
  # rejects "" as a fallback too — it only accepts non-null, NON-EMPTY-STRING
  # arguments, so an all-empty/null argument list is a hard plan error, not a
  # graceful "". Confirmed live once DEV_TFVARS/credentials were configured
  # (PR #32 comment: "Fix error in terraform plan dev") — the ternary below
  # doesn't have that restriction.
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
