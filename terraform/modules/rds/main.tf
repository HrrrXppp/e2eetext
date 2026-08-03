# Phase 3 of #27: the RDS PostgreSQL instance, its subnet group, security
# group, and a parameter group carrying the pg_cron settings from #20's plan
# (shared_preload_libraries=pg_cron, cron.database_name) — folding that
# manual "create/modify a parameter group, associate it, reboot" step into
# Terraform, per the approved Phase 3 plan on #27.
#
# Like the ec2 module before it, this module's job is to describe an
# ALREADY-EXISTING, hand-created RDS instance precisely enough for
# `terraform import` — not to invent sizing/engine defaults. Every
# identifying/config attribute below has no default on purpose: a guessed
# value that doesn't match the live instance either forces a replace
# (ForceNew attributes: engine, db_name, username, storage_encrypted, ...)
# or, at minimum, leaves `terraform plan` non-zero-diff, which is the
# acceptance bar for every phase of #27. See README's Terraform section for
# how to read the real values off the live instance before importing.

locals {
  create_security_group = length(var.security_group_ids) == 0
  security_group_ids    = local.create_security_group ? [aws_security_group.this[0].id] : var.security_group_ids

  # App PostgreSQL database (DATABASE_URL / cron.database_name). Distinct
  # from CreateDBInstance DBName (var.db_name), which is ForceNew and often
  # null on brownfield imports even when a later CREATE DATABASE exists.
  # Priority: explicit cron override → app_database_name → db_name → "messenger".
  app_database_name_effective = coalesce(var.app_database_name, var.db_name, "messenger")
  cron_database_name          = var.cron_database_name != "" ? var.cron_database_name : local.app_database_name_effective

  final_snapshot_identifier      = var.final_snapshot_identifier != "" ? var.final_snapshot_identifier : "${var.identifier}-final"
  parameter_group_name_effective = var.create_parameter_group ? aws_db_parameter_group.this[0].name : var.parameter_group_name
  subnet_group_name_effective    = var.create_subnet_group ? aws_db_subnet_group.this[0].name : var.subnet_group_name
}

# Not data-source-only like modules/network (an RDS instance is far more
# consequential to get wrong than a VPC lookup, and the whole point of this
# module is to bring it under management) — but subnet group membership
# follows the same "describe what's real" spirit: pass the live instance's
# actual subnets, not a guess.
#
# count, not unconditional, because the AWS-implicit "default" DB subnet
# group (every default VPC has one) can't be created via this resource — the
# provider rejects `name = "default"` outright ("Default" is not allowed as
# "name", PR #32 comment: real terraform plan (dev) failure once live
# credentials/tfvars were configured, dev's live instance sits in that
# implicit default group). Set create_subnet_group = false to reference an
# existing group (default or otherwise) by name without trying to create it.
resource "aws_db_subnet_group" "this" {
  count = var.create_subnet_group ? 1 : 0

  name        = var.subnet_group_name
  subnet_ids  = var.subnet_ids
  description = "e2eetext RDS subnet group"

  tags = var.tags
}

resource "aws_security_group" "this" {
  count = local.create_security_group ? 1 : 0

  name        = var.security_group_name
  description = "e2eetext RDS instance: PostgreSQL (5432) from the app tier only"
  vpc_id      = var.vpc_id

  # If the live instance's real SG was hand-made (as Phase 2 found for the
  # dev EC2 instance's "launch-wizard-2"/"ec2-rds-1" SGs), pass its ID via
  # var.security_group_ids instead of creating a new one here — swapping a
  # live RDS instance's SG is a real firewall change, not a no-op.
  dynamic "ingress" {
    for_each = var.ingress_security_group_id != "" ? [1] : []
    content {
      description     = "PostgreSQL from the app security group"
      from_port       = 5432
      to_port         = 5432
      protocol        = "tcp"
      security_groups = [var.ingress_security_group_id]
    }
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = var.tags

  lifecycle {
    # PR #32 review: on dev, module.ec2's own security_group_id output is
    # null (dev's EC2 instance uses pre-existing, hand-made SGs), so an
    # unset var.ingress_security_group_id combined with an empty
    # var.security_group_ids silently creates this SG with ZERO ingress
    # rules — an accidental apply would cut the app tier off from its own
    # database instead of failing loudly. Require the ingress SG explicitly
    # whenever this module is the one creating the security group.
    precondition {
      condition     = var.ingress_security_group_id != ""
      error_message = "modules/rds is creating its own security group (var.security_group_ids is empty) but var.ingress_security_group_id is unset — that would create an RDS security group with no ingress rule at all. Set ingress_security_group_id explicitly (e.g. the app tier's SG id), or pass pre-existing security_group_ids instead of letting this module create one."
    }
  }
}

# #20's plan: "create or modify a custom DB parameter group with
# shared_preload_libraries=pg_cron (and cron.database_name=messenger if
# settable)". Both are static parameters (apply_method = "pending-reboot")
# — per #20's plan, the instance must be rebooted for them to take effect,
# and that must happen before any migration runs. This module does not
# reboot the instance automatically; see README for the documented
# post-apply reboot step.
resource "aws_db_parameter_group" "this" {
  count = var.create_parameter_group ? 1 : 0

  name        = var.parameter_group_name
  family      = var.parameter_group_family
  description = "e2eetext RDS parameter group${var.enable_pg_cron ? " (pg_cron enabled per #20)" : ""}"

  # PR #32 review: shared_preload_libraries is a full replace, not an
  # append — if the live parameter group already preloads other libraries,
  # managing this group with a hardcoded "pg_cron" would silently drop
  # them on apply. var.shared_preload_libraries defaults to just
  # ["pg_cron"] (this module's own scope, #20's plan) but let brownfield
  # imports list the live instance's complete preload set instead.
  dynamic "parameter" {
    for_each = var.enable_pg_cron ? [1] : []
    content {
      name         = "shared_preload_libraries"
      value        = join(",", var.shared_preload_libraries)
      apply_method = "pending-reboot"
    }
  }

  dynamic "parameter" {
    for_each = var.enable_pg_cron ? [1] : []
    content {
      name         = "cron.database_name"
      value        = local.cron_database_name
      apply_method = "pending-reboot"
    }
  }

  tags = var.tags

  lifecycle {
    # A DB parameter group in use by an instance can't be deleted; creating
    # its replacement first (on a name/family change) avoids a needless
    # apply-time failure.
    create_before_destroy = true
  }
}

resource "aws_db_instance" "this" {
  identifier = var.identifier

  engine         = var.engine
  engine_version = var.engine_version
  instance_class = var.instance_class

  allocated_storage = var.allocated_storage
  storage_type      = var.storage_type
  storage_encrypted = var.storage_encrypted
  # gp3-only knobs and a customer KMS key — all null (provider/AWS default:
  # "not set" / account default key) unless the live instance actually uses
  # them. PR #32 review: these were previously absent entirely, which is
  # fine for gp2 default-KMS instances but leaves plan noise/ForceNew risk
  # for anything using gp3 IOPS/throughput or a custom key — now they're
  # available to set for a zero-diff import instead of being unmanageable.
  iops               = var.iops
  storage_throughput = var.storage_throughput
  kms_key_id         = var.kms_key_id
  # Autoscaling ceiling for storage. 0 (this module's default) matches
  # AWS's own "disabled" default — only meaningfully diverges from live
  # once someone sets it, same "describe reality" rule as the required
  # block above.
  max_allocated_storage = var.max_allocated_storage

  db_name  = var.db_name
  username = var.username
  password = var.password

  db_subnet_group_name   = local.subnet_group_name_effective
  vpc_security_group_ids = local.security_group_ids
  parameter_group_name   = local.parameter_group_name_effective
  ca_cert_identifier     = var.ca_cert_identifier

  multi_az                   = var.multi_az
  publicly_accessible        = var.publicly_accessible
  backup_retention_period    = var.backup_retention_period
  backup_window              = var.backup_window
  maintenance_window         = var.maintenance_window
  auto_minor_version_upgrade = var.auto_minor_version_upgrade
  apply_immediately          = var.apply_immediately
  deletion_protection        = var.deletion_protection
  copy_tags_to_snapshot      = var.copy_tags_to_snapshot
  network_type               = var.network_type

  # Enhanced monitoring / Performance Insights — both off (AWS defaults) in
  # this module's own defaults; set to match the live instance before
  # import rather than leaving them as post-import plan noise (PR #32
  # review: "Incomplete aws_db_instance attrs -> post-import drift risk").
  monitoring_interval                   = var.monitoring_interval
  monitoring_role_arn                   = var.monitoring_role_arn
  performance_insights_enabled          = var.performance_insights_enabled
  performance_insights_kms_key_id       = var.performance_insights_kms_key_id
  performance_insights_retention_period = var.performance_insights_enabled ? var.performance_insights_retention_period : null

  enabled_cloudwatch_logs_exports = var.enabled_cloudwatch_logs_exports

  skip_final_snapshot       = var.skip_final_snapshot
  final_snapshot_identifier = var.skip_final_snapshot ? null : local.final_snapshot_identifier

  tags = var.tags

  lifecycle {
    # AWS never returns the master password on read, so Terraform can never
    # confirm var.password matches what's actually live — without this,
    # every plan against an imported instance would show a permanent
    # password diff (or worse, `apply` would rotate the live password).
    # Rotate the real password out-of-band (console/CLI, or a
    # Secrets-Manager-managed rotation) if it ever needs to change; update
    # var.password to match afterwards so this module's tfvars stay
    # informational/consistent even though Terraform itself never acts on
    # it.
    ignore_changes = [password]
  }
}
