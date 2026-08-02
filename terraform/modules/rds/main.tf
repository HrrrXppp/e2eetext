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

  # cron.database_name defaults to the app's own database (var.db_name) —
  # issue #20's plan calls out "cron.database_name=messenger if settable",
  # and messenger is exactly what var.db_name is expected to be, so this
  # avoids requiring the same value twice.
  cron_database_name = var.cron_database_name != "" ? var.cron_database_name : var.db_name

  final_snapshot_identifier      = var.final_snapshot_identifier != "" ? var.final_snapshot_identifier : "${var.identifier}-final"
  parameter_group_name_effective = var.create_parameter_group ? aws_db_parameter_group.this[0].name : var.parameter_group_name
}

# Not data-source-only like modules/network (an RDS instance is far more
# consequential to get wrong than a VPC lookup, and the whole point of this
# module is to bring it under management) — but subnet group membership
# follows the same "describe what's real" spirit: pass the live instance's
# actual subnets, not a guess.
resource "aws_db_subnet_group" "this" {
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

  dynamic "parameter" {
    for_each = var.enable_pg_cron ? [1] : []
    content {
      name         = "shared_preload_libraries"
      value        = "pg_cron"
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

  db_name  = var.db_name
  username = var.username
  password = var.password

  db_subnet_group_name   = aws_db_subnet_group.this.name
  vpc_security_group_ids = local.security_group_ids
  parameter_group_name   = local.parameter_group_name_effective

  multi_az                   = var.multi_az
  publicly_accessible        = var.publicly_accessible
  backup_retention_period    = var.backup_retention_period
  backup_window              = var.backup_window
  maintenance_window         = var.maintenance_window
  auto_minor_version_upgrade = var.auto_minor_version_upgrade
  apply_immediately          = var.apply_immediately
  deletion_protection        = var.deletion_protection

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
