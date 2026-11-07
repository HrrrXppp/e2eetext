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

  admin_ssh_cidr_blocks = var.admin_ssh_cidr_blocks
  key_name              = var.key_name

  # Live SGs stay imported; IAM instance profile + boot deploy are owned by
  # Terraform (defaults true) so dig pull/start is not a manual host step.
  security_group_ids          = local.ec2_security_group_ids
  create_iam_instance_profile = var.create_ec2_iam_instance_profile

  security_group_name       = var.ec2_security_group_name
  iam_role_name             = var.ec2_iam_role_name
  iam_instance_profile_name = var.ec2_iam_instance_profile_name

  manage_boot_deploy    = var.ec2_manage_boot_deploy
  boot_deploy_repo_root = var.ec2_boot_deploy_repo_root
  boot_deploy_image_tag = local.deployed_version # pinned in version.tf
  boot_deploy_start_now = var.ec2_boot_deploy_start_now

  # #37: lets the server publish WebSocketConnections to CloudWatch.
  metrics_namespace = var.metrics_namespace

  tags = var.tags
}

# module.alb is gone. The next apply destroys dev-e2eetext, its target
# groups, listeners, and rules. The pre-existing ALB security group is not
# managed here and stays in the account.

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

  db_name           = var.rds_db_name
  app_database_name = var.rds_app_database_name
  username          = var.rds_username
  password          = var.rds_password

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

# --- Metrics (#37) ---
# Scheduled Lambda publishing Messages/Chats/Users to CloudWatch. Skipped
# unless metrics_enabled is true and metrics_database_url_secret_arn is a
# Secrets Manager ARN. An empty ARN must not be planned: the Lambda IAM
# policy would send Resource "" and AWS would reject it. Also needs the
# package from lambdas/metrics/build.sh. No public endpoint.
check "metrics_database_secret" {
  assert {
    condition     = !var.metrics_enabled || trimspace(var.metrics_database_url_secret_arn) != ""
    error_message = "metrics_enabled is true but metrics_database_url_secret_arn is empty, so the metrics Lambda is not created. Set the Secrets Manager ARN, or set metrics_enabled = false."
  }
}

module "metrics" {
  source = "../../modules/metrics"
  count  = var.metrics_enabled && trimspace(var.metrics_database_url_secret_arn) != "" ? 1 : 0

  name        = var.name_prefix
  environment = "dev"
  namespace   = var.metrics_namespace

  lambda_zip_path         = "${path.module}/../../../lambdas/metrics/build/metrics.zip"
  database_url_secret_arn = var.metrics_database_url_secret_arn

  vpc_id     = module.network.vpc_id
  subnet_ids = length(var.metrics_subnet_ids) > 0 ? var.metrics_subnet_ids : [var.ec2_subnet_id != "" ? var.ec2_subnet_id : module.network.subnet_ids[0]]

  rds_security_group_id = var.metrics_rds_security_group_id != "" ? var.metrics_rds_security_group_id : (module.rds.security_group_id != null ? module.rds.security_group_id : "")
  create_vpc_endpoints  = var.metrics_create_vpc_endpoints
  # Dev EC2 keeps hand-made SGs (launch-wizard-2, ec2-rds-1). The module
  # does not create one, so security_group_id is null — pass the resolved IDs.
  endpoint_ingress_security_group_ids = distinct(concat(
    local.ec2_security_group_ids,
    module.ec2.security_group_id != null ? [module.ec2.security_group_id] : [],
  ))

  tags = var.tags
}

# These 443 permissions already exist on the dig endpoint group (Lambda,
# launch-wizard-2, ec2-rds-1). Creating them again returns
# InvalidPermission.Duplicate. Import adopts each one into
# aws_security_group_rule so apply does not call AuthorizeSecurityGroupIngress.
# Lookups stay outside module.metrics: an import id that reads that module
# cycles with the rule it is importing.
data "aws_security_group" "metrics_endpoints" {
  name   = "${var.name_prefix}-metrics-endpoints"
  vpc_id = module.network.vpc_id
}

data "aws_security_group" "metrics_lambda" {
  name   = "${var.name_prefix}-metrics-lambda"
  vpc_id = module.network.vpc_id
}

import {
  to = module.metrics[0].aws_security_group_rule.endpoints_https[0]
  id = "${data.aws_security_group.metrics_endpoints.id}_ingress_tcp_443_443_${data.aws_security_group.metrics_lambda.id}"
}

import {
  to = module.metrics[0].aws_security_group_rule.endpoints_https[1]
  id = "${data.aws_security_group.metrics_endpoints.id}_ingress_tcp_443_443_${local.ec2_security_group_ids[0]}"
}

import {
  to = module.metrics[0].aws_security_group_rule.endpoints_https[2]
  id = "${data.aws_security_group.metrics_endpoints.id}_ingress_tcp_443_443_${local.ec2_security_group_ids[1]}"
}
