# --- Identity / core config ---
# No defaults on this block, deliberately (same pattern as modules/ec2's
# ami_id/instance_type): these either identify the live instance for
# `terraform import` or are ForceNew/high-blast-radius attributes where a
# guessed value risks a forced replace of live infrastructure. Read the
# real values off the live instance first — see README's Terraform section.

variable "identifier" {
  description = "DB instance identifier of the live RDS instance. Required for `terraform import` (ForceNew if it ever needs to change)."
  type        = string
}

variable "engine" {
  description = "RDS engine. Defaults to \"postgres\" — the only engine this app has ever used (README: \"Database | PostgreSQL 16\"), not a guess about a specific live instance's config."
  type        = string
  default     = "postgres"
}

variable "engine_version" {
  description = <<-EOT
    Engine version of the live instance (e.g. "16.4"). No default: RDS
    PostgreSQL minor versions are region/time specific, and a mismatch
    causes an in-place version-change diff on plan at best, or a
    replace/downtime at worst. Get the real value with `aws rds
    describe-db-instances --db-instance-identifier <id> --query
    'DBInstances[0].EngineVersion'` before importing.
  EOT
  type        = string
}

variable "instance_class" {
  description = "Instance class of the live instance (e.g. \"db.t3.micro\"). Must match exactly for a zero-diff import."
  type        = string
}

variable "allocated_storage" {
  description = "Allocated storage (GiB) of the live instance. Must match exactly for a zero-diff import (a mismatch triggers a real storage-modify apply, not just a plan diff)."
  type        = number
}

variable "storage_type" {
  description = "Storage type of the live instance (e.g. \"gp2\", \"gp3\"). Must match the live value."
  type        = string
}

variable "storage_encrypted" {
  description = "Whether the live instance's storage is encrypted at rest. ForceNew if it ever needs to change (can't be toggled in place) — must match the live value exactly."
  type        = bool
}

variable "db_name" {
  description = "Initial database name (README: \"messenger\"). ForceNew — ignored by AWS on import (RDS doesn't expose the original CREATE DATABASE name for re-verification), but must still be set to avoid Terraform proposing to recreate the database on a future replace."
  type        = string
}

variable "username" {
  description = "Master username of the live instance. ForceNew — must match the live value exactly."
  type        = string
}

variable "password" {
  description = <<-EOT
    Master password. Never hardcode this in .tfvars — source it from AWS
    Secrets Manager / SSM Parameter Store (matching how server/db
    credentials are already kept out of version control elsewhere in this
    repo) and pass it via -var or TF_VAR_password at plan/apply time. This
    module ignores changes to it after creation (see main.tf) since AWS
    never returns the real value for comparison, so Terraform can't verify
    a match anyway — rotate the live password out-of-band if needed.
  EOT
  type        = string
  sensitive   = true
}

# --- Networking ---

variable "vpc_id" {
  description = "VPC ID for the instance's security group, when this module creates one (module.network's output)."
  type        = string
}

variable "subnet_ids" {
  description = <<-EOT
    Subnet IDs for the DB subnet group. No default: unlike the ALB/EC2
    subnets (which are the account's public default-VPC subnets per
    Phase 2's audit), RDS is commonly placed in private subnets, and
    nothing in this repo confirms which is true here. Get the real value
    with `aws rds describe-db-subnet-groups` before importing.
  EOT
  type        = list(string)
}

variable "subnet_group_name" {
  description = "Name of the DB subnet group. Must match the live subnet group's name for a zero-diff import."
  type        = string
}

variable "security_group_ids" {
  description = <<-EOT
    Pre-existing security group IDs to attach to the instance instead of
    creating one. Leave empty (default) to have this module create its own
    SG (named var.security_group_name, allowing 5432 from
    var.ingress_security_group_id). Phase 2 found the live dev EC2
    instance runs with hand-made SGs including one named "ec2-rds-1" whose
    own description warns that detaching it can cut RDS connectivity —
    that same SG (or its RDS-side counterpart) is a likely candidate here.
    Set explicitly to whatever the live instance's real SGs are for a
    zero-diff import.
  EOT
  type        = list(string)
  default     = []
}

variable "security_group_name" {
  description = "Name for the instance's security group, if this module creates one (security_group_ids empty). Must match the live SG's name for a zero-diff import."
  type        = string
  default     = "e2eetext-rds"
}

variable "ingress_security_group_id" {
  description = <<-EOT
    Security group ID allowed to reach port 5432 on the created SG (the
    EC2 instance's SG — module.ec2's security_group_id output, or the
    live "ec2-rds-1"-equivalent group). Only used when this module creates
    its own SG (security_group_ids empty). Leave empty to create the SG
    with no ingress rule at all (not useful in practice — set this).
  EOT
  type        = string
  default     = ""
}

# --- Parameter group / pg_cron (#20) ---

variable "create_parameter_group" {
  description = "Whether this module creates/manages the DB parameter group. Set false to reference an already-existing parameter group by name (var.parameter_group_name) without managing its parameters."
  type        = bool
  default     = true
}

variable "parameter_group_name" {
  description = <<-EOT
    Name of the DB parameter group. If create_parameter_group is true and
    no such group exists yet, this is the name it's created with. If one
    already exists live (e.g. from following #20's manual steps by hand
    before this module existed), this must match its real name for a
    zero-diff import.
  EOT
  type        = string
}

variable "parameter_group_family" {
  description = <<-EOT
    Parameter group family, e.g. "postgres16" for engine_version 16.x.
    Must match var.engine_version's major version exactly — AWS rejects
    apply with InvalidParameterCombination otherwise. No default since it
    depends on engine_version, which itself has no default.
  EOT
  type        = string
}

variable "enable_pg_cron" {
  description = <<-EOT
    Whether to set shared_preload_libraries=pg_cron and
    cron.database_name on the parameter group, per #20's plan
    ("Disappearing messages: ensure RDS allows pg_cron before migrate").
    Both are static parameters (apply_method "pending-reboot") — per #20's
    plan, the instance needs a reboot after apply for them to take effect,
    and that must happen before migration 000006 (which does
    `CREATE EXTENSION pg_cron`) ever runs against this instance. This
    module does not reboot the instance automatically; see README.
  EOT
  type        = bool
  default     = true
}

variable "cron_database_name" {
  description = "Value for the parameter group's cron.database_name (only set when enable_pg_cron is true). Leave empty (default) to reuse var.db_name — issue #20's plan expects this to equal the app's own database, \"messenger\"."
  type        = string
  default     = ""
}

# --- Safety / operational defaults ---
# Unlike the block above, these are genuine policy choices documented
# elsewhere in this repo (or conservative, non-guessy safe defaults), not
# live-instance facts — so they do have defaults, same spirit as the ec2
# module hardcoding its security group's port list in main.tf rather than
# exposing it as an unguessable variable.

variable "publicly_accessible" {
  description = "Whether the instance has a public IP. Defaults to false — README's AWS walkthrough is explicit: \"do not expose PostgreSQL to the public internet\"."
  type        = bool
  default     = false
}

variable "multi_az" {
  description = "Whether the instance is Multi-AZ. No default: this is a real cost/availability trade-off per environment, not a guessable fact — confirm against the live instance (`aws rds describe-db-instances --query 'DBInstances[0].MultiAZ'`) before importing, or decide deliberately if applying fresh."
  type        = bool
}

variable "backup_retention_period" {
  description = "Automated backup retention in days. No default: 0 (disabled) vs. a real retention window is a meaningful operational choice, and a mismatch against the live instance still breaks the zero-diff import bar even though it isn't ForceNew."
  type        = number
}

variable "backup_window" {
  description = "Preferred backup window (e.g. \"03:00-04:00\", UTC). Must match the live instance's value for a zero-diff import."
  type        = string
}

variable "maintenance_window" {
  description = "Preferred maintenance window (e.g. \"sun:04:30-sun:05:30\", UTC). Must match the live instance's value for a zero-diff import."
  type        = string
}

variable "auto_minor_version_upgrade" {
  description = "Whether RDS may auto-apply minor engine version upgrades. Defaults to true (the AWS default) — if this causes engine_version drift against var.engine_version in future plans, either pin it false or bump engine_version to match."
  type        = bool
  default     = true
}

variable "apply_immediately" {
  description = "Whether instance modifications apply immediately instead of during the next maintenance window. Defaults to false — surprise immediate changes to a live database are exactly the kind of risk #27's phased, zero-diff-first approach is meant to avoid."
  type        = bool
  default     = false
}

variable "deletion_protection" {
  description = "Whether the instance is protected from `terraform destroy` / console deletion. Defaults to true as a safety net; override to false only deliberately (e.g. a genuinely disposable dev/test instance)."
  type        = bool
  default     = true
}

variable "skip_final_snapshot" {
  description = "Whether to skip taking a final snapshot on destroy. Defaults to false (always snapshot) as a safety net for a database holding real user data."
  type        = bool
  default     = false
}

variable "final_snapshot_identifier" {
  description = "Final snapshot identifier, used when skip_final_snapshot is false. Leave empty (default) to derive \"<identifier>-final\"."
  type        = string
  default     = ""
}

variable "tags" {
  description = "Tags applied to the DB instance, subnet group, security group (if created), and parameter group (if created)."
  type        = map(string)
  default     = {}
}
