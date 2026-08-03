variable "aws_region" {
  description = "AWS region for dev's networking/ALB/EC2 resources."
  type        = string
  default     = "us-east-2"
}

variable "name_prefix" {
  description = <<-EOT
    Name tag for the EC2 instance. Default "" because the live dev instance
    carries no tags at all (confirmed 2026-07-26) and the import acceptance
    bar is a zero-diff plan. Set to e.g. "e2eetext-dev" as a deliberate
    post-import change to give the instance a Name tag.
  EOT
  type        = string
  default     = ""
}

# Defaults below marked "confirmed live" were read from the real dev stack
# via the AWS API on 2026-07-26 — they are recorded facts, not guesses (the
# earlier no-defaults design guarded against guessed values forcing a
# replace of live resources on `apply`; confirmed values are exactly what
# that design was waiting for). With them, `terraform plan` runs with no
# terraform.tfvars at all. Identifiers that would expose account-specific
# IDs/ARNs in this public repo stay out of the defaults — main.tf looks
# those up from the live stack by name instead.

# --- Networking ---

variable "vpc_id" {
  description = "VPC ID the live dev stack runs in. Leave empty to use the account's default VPC (confirmed live: dev runs in the default VPC)."
  type        = string
  default     = ""
}

variable "subnet_ids" {
  description = "Subnet IDs to consider for placement. Leave empty to use every subnet in the selected VPC (confirmed live: the dev ALB spans all three default-VPC subnets)."
  type        = list(string)
  default     = []
}

variable "alb_subnet_ids" {
  description = "Public subnet IDs for the ALB specifically. Leave empty to reuse module.network's subnet_ids."
  type        = list(string)
  default     = []
}

# --- EC2 ---

variable "ami_id" {
  description = "AMI ID of the live dev EC2 instance. Default is the confirmed live value (2026-07-26)."
  type        = string
  default     = "ami-0e5497a77ef21b5ac"
}

variable "instance_type" {
  description = "Instance type of the live dev EC2 instance. Default is the confirmed live value (2026-07-26)."
  type        = string
  default     = "t3.small"
}

variable "ec2_subnet_id" {
  description = <<-EOT
    Subnet ID the dev EC2 instance actually launches in. Leave empty to use
    the first subnet from module.network — fine for a plan, but before
    `terraform import` set this to the live instance's real SubnetId (`aws
    ec2 describe-instances`): module.network's subnet order isn't
    guaranteed, and a subnet_id mismatch forces a replace.
  EOT
  type        = string
  default     = ""
}

variable "key_name" {
  description = "EC2 key pair name of the live dev instance. Default is the confirmed live value (2026-07-26)."
  type        = string
  default     = "alpha-0.0.1"
}

variable "ec2_security_group_names" {
  description = <<-EOT
    Names of the pre-existing security groups attached to the live dev
    instance, resolved to IDs in main.tf. Defaults are the confirmed live
    values (2026-07-26). NOTE: neither is the purpose-built
    "e2eetext-dev-ec2" SG the ec2 module would create — dev's SGs were made
    by hand ("launch-wizard-2") and by the RDS console ("ec2-rds-1", which
    warns that detaching it can cut RDS connectivity). Set to [] (with
    ec2_security_group_ids also []) to have the module create its own SG
    instead — a deliberate firewall migration, not an import.
  EOT
  type        = list(string)
  default     = ["launch-wizard-2", "ec2-rds-1"]
}

variable "ec2_security_group_ids" {
  description = "Explicit security group IDs for the instance, bypassing the by-name lookup of ec2_security_group_names. Leave empty to use the lookup."
  type        = list(string)
  default     = []
}

variable "create_ec2_iam_instance_profile" {
  description = <<-EOT
    Whether to create/attach the ECR-read IAM role + instance profile.
    Default false because the live dev instance runs with NO instance
    profile at all (confirmed 2026-07-26 — the account has zero instance
    profiles, despite the README's documented setup). Flip to true to
    migrate dev to the documented ECR-pull-via-instance-profile setup (an
    in-place instance update).
  EOT
  type        = bool
  default     = false
}

variable "admin_ssh_cidr_blocks" {
  description = "CIDR blocks allowed to SSH to the dev instance. Only used when this config creates the instance SG (it doesn't for live dev)."
  type        = list(string)
  default     = []
}

variable "ec2_security_group_name" {
  description = "Name for the instance SG, if this config is ever asked to create one (ec2_security_group_names/_ids both empty)."
  type        = string
  default     = "e2eetext-dev-ec2"
}

variable "ec2_iam_role_name" {
  description = "Name for the instance IAM role, if create_ec2_iam_instance_profile is flipped to true."
  type        = string
  default     = "e2eetext-dev-ec2-role"
}

variable "ec2_iam_instance_profile_name" {
  description = "Name for the instance profile, if create_ec2_iam_instance_profile is flipped to true."
  type        = string
  default     = "e2eetext-dev-ec2-profile"
}

# --- ALB ---
# The live dev ALB was built by hand and diverges from
# create-alb.example.sh in name, target-group names, idle timeout, default
# action, and rule layout. All defaults below are the confirmed live values
# (2026-07-26) so plan/import converge on reality; see modules/alb for what
# the script-shaped defaults would be.

variable "alb_name" {
  description = "Name of the live dev ALB (confirmed live: \"dev-e2eetext\", not \"e2eetext-dev\"). Also the lookup key for the ALB SG / ACM cert data sources in main.tf. ForceNew on the ALB."
  type        = string
  default     = "dev-e2eetext"
}

variable "server_target_group_name" {
  description = "Name of the live server (8081) target group (confirmed live: \"dev-server\"). ForceNew — a mismatch means TG replacement, i.e. downtime."
  type        = string
  default     = "dev-server"
}

variable "client_target_group_name" {
  description = "Name of the live client (8080) target group (confirmed live: \"client\"). Same ForceNew caveat."
  type        = string
  default     = "client"
}

variable "alb_idle_timeout" {
  description = "ALB idle timeout in seconds (confirmed live: 60 — the AWS default; create-alb.example.sh's 3600-for-websockets was never applied to dev)."
  type        = number
  default     = 60
}

variable "https_default_action" {
  description = "HTTPS listener default action shape (confirmed live: \"fixed-404\" — dev's client routing is hand-made path rules, not a catch-all forward)."
  type        = string
  default     = "fixed-404"
}

variable "api_rule_priority" {
  description = "Priority of the /api listener rule (confirmed live: 100; priorities 10-14 are taken by hand-made client rules Terraform doesn't manage yet)."
  type        = number
  default     = 100
}

variable "api_path_patterns" {
  description = "Path patterns of the /api listener rule (confirmed live: [\"/api/*\"])."
  type        = list(string)
  default     = ["/api/*"]
}

variable "health_check_healthy_threshold" {
  description = "TG healthy threshold (confirmed live: 5 on both dev TGs, not the provider default of 3)."
  type        = number
  default     = 5
}

variable "health_check_unhealthy_threshold" {
  description = "TG unhealthy threshold (confirmed live: 2 on both dev TGs, not the provider default of 3)."
  type        = number
  default     = 2
}

variable "server_health_check_port" {
  description = "Server TG health-check port (confirmed live: pinned to \"8081\", not \"traffic-port\" — functionally identical, but a diff if mismatched)."
  type        = string
  default     = "8081"
}

variable "client_health_check_port" {
  description = "Client TG health-check port (confirmed live: pinned to \"8080\")."
  type        = string
  default     = "8080"
}

variable "create_health_rule" {
  description = "Whether to add the script's /health -> server listener rule (confirmed live: absent on dev; target-group health checks hit /health directly and need no rule)."
  type        = bool
  default     = false
}

variable "alb_security_group_id" {
  description = "Security group ID of the dev ALB. Leave empty to look it up from the live ALB by name (its SG is named \"dev-lb\"); set explicitly only if the ALB doesn't exist yet."
  type        = string
  default     = ""
}

variable "acm_certificate_arn" {
  description = "ACM certificate ARN for the dev ALB's HTTPS listener. Leave empty to read it off the live ALB's 443 listener; set explicitly only if the ALB doesn't exist yet."
  type        = string
  default     = ""
}

variable "tags" {
  description = <<-EOT
    Tags applied to all dev resources. Default {} because the live,
    hand-made dev stack is untagged (confirmed 2026-07-26) and the import
    acceptance bar is a zero-diff plan. Add e.g.
    { Environment = "dev", ManagedBy = "terraform" } as a deliberate
    post-import apply once the import is clean.
  EOT
  type        = map(string)
  default     = {}
}

# --- RDS (Phase 3 of #27) ---
# No confirmed-live defaults here (see main.tf's rds module header) — this
# phase was built without AWS credentials. Required variables below have no
# default on purpose, the same "describe reality, don't guess" rule
# modules/rds and modules/ec2 both follow. Fill in terraform.tfvars from a
# real AWS audit before planning or importing.

variable "rds_identifier" {
  description = "DB instance identifier of the live dev RDS instance. Get with `aws rds describe-db-instances --query 'DBInstances[].DBInstanceIdentifier'`."
  type        = string
}

variable "rds_engine_version" {
  description = "Engine version of the live dev RDS instance (e.g. \"16.4\")."
  type        = string
}

variable "rds_instance_class" {
  description = "Instance class of the live dev RDS instance (e.g. \"db.t3.micro\")."
  type        = string
}

variable "rds_allocated_storage" {
  description = "Allocated storage (GiB) of the live dev RDS instance."
  type        = number
}

variable "rds_storage_type" {
  description = "Storage type of the live dev RDS instance (e.g. \"gp2\", \"gp3\")."
  type        = string
}

variable "rds_storage_encrypted" {
  description = "Whether the live dev RDS instance's storage is encrypted at rest."
  type        = bool
}

variable "rds_db_name" {
  description = "Initial database name. Default \"messenger\" — the fixed value used throughout this repo (docker-compose, README's connection string example), not a guess about a specific live instance."
  type        = string
  default     = "messenger"
}

variable "rds_username" {
  description = "Master username of the live dev RDS instance. No default: README's AWS walkthrough treats this as chosen at creation time, not a fixed convention."
  type        = string
}

variable "rds_password" {
  description = "Master password of the live dev RDS instance. Never commit this — pass via TF_VAR_rds_password or -var, sourced from Secrets Manager/SSM, not terraform.tfvars."
  type        = string
  sensitive   = true
}

variable "rds_subnet_ids" {
  description = <<-EOT
    Subnet IDs for the DB subnet group. No default (PR #32 review): this
    used to fall back to module.network's subnets when unset, but those
    are the account's public default-VPC subnets (confirmed live for the
    ALB/EC2 in Phase 2) — RDS is commonly placed in PRIVATE subnets
    instead, and silently defaulting to public ones risked pairing a
    future publicly_accessible mistake with an already-public subnet
    group. Get the live instance's real subnet-group members with
    `aws rds describe-db-subnet-groups` before planning or importing.
  EOT
  type        = list(string)
}

variable "rds_subnet_group_name" {
  description = "Name of the live dev DB subnet group. Must match exactly for a zero-diff import."
  type        = string
}

variable "rds_create_subnet_group" {
  description = <<-EOT
    Whether the rds module creates/manages the DB subnet group. Default
    true; set false when rds_subnet_group_name is a pre-existing group this
    module shouldn't try to create — including the AWS-implicit "default"
    DB subnet group every default VPC has, which the provider refuses to
    create (name "default" is reserved). Confirmed needed live for dev
    (PR #32 comment: "Fix error in terraform plan dev").
  EOT
  type        = bool
  default     = true
}

variable "rds_security_group_ids" {
  description = "Pre-existing security group IDs for the dev RDS instance (skips creating one). Leave empty to have the rds module create its own SG. If the live instance uses hand-made SGs (as dev's EC2 instance does — \"ec2-rds-1\" per Phase 2's audit), set the real ID(s) here for a zero-diff import."
  type        = list(string)
  default     = []
}

variable "rds_security_group_name" {
  description = "Name for the dev RDS security group, if the rds module creates one (rds_security_group_ids empty)."
  type        = string
  default     = "e2eetext-dev-rds"
}

variable "rds_ingress_security_group_id" {
  description = "Security group ID allowed to reach 5432 on the created RDS SG. Leave empty to fall back to module.ec2's own SG (null for dev, since dev's EC2 instance uses pre-existing SGs — see ec2_security_group_names) — set explicitly (e.g. to \"ec2-rds-1\"'s real ID) before import."
  type        = string
  default     = ""
}

variable "rds_create_parameter_group" {
  description = "Whether the rds module creates/manages the DB parameter group. Default true so Phase 3 folds in #20's pg_cron settings; set false to reference an already-existing one by name without managing its parameters."
  type        = bool
  default     = true
}

variable "rds_parameter_group_name" {
  description = "Name of the dev DB parameter group — created fresh with this name if rds_create_parameter_group is true and none exists yet, or must match a real live name otherwise."
  type        = string
  default     = "e2eetext-dev-rds-pgcron"
}

variable "rds_parameter_group_family" {
  description = "Parameter group family. Default \"postgres16\" per README's fixed tech-stack version (\"Database | PostgreSQL 16\") — must match rds_engine_version's major version exactly (no default there, since the minor version is unconfirmed)."
  type        = string
  default     = "postgres16"
}

variable "rds_enable_pg_cron" {
  description = "Whether to set shared_preload_libraries=pg_cron + cron.database_name on the parameter group, per #20's plan. Default true — folding #20's manual pg_cron step into Terraform is exactly Phase 3's scope."
  type        = bool
  default     = true
}

variable "rds_cron_database_name" {
  description = "Value for cron.database_name. Leave empty (default) to reuse rds_db_name."
  type        = string
  default     = ""
}

variable "rds_multi_az" {
  description = "Whether the live dev RDS instance is Multi-AZ. No default — confirm against the live instance before importing."
  type        = bool
}

variable "rds_publicly_accessible" {
  description = "Whether the dev RDS instance has a public IP. Default false — README's AWS walkthrough is explicit that PostgreSQL must not be exposed to the public internet."
  type        = bool
  default     = false
}

variable "rds_backup_retention_period" {
  description = "Automated backup retention in days for the live dev RDS instance. No default — confirm the real value before importing (0 = disabled)."
  type        = number
}

variable "rds_backup_window" {
  description = "Preferred backup window (UTC) of the live dev RDS instance, e.g. \"03:00-04:00\". No default — must match the live value."
  type        = string
}

variable "rds_maintenance_window" {
  description = "Preferred maintenance window (UTC) of the live dev RDS instance, e.g. \"sun:04:30-sun:05:30\". No default — must match the live value."
  type        = string
}

variable "rds_deletion_protection" {
  description = "Whether the dev RDS instance is protected from `terraform destroy` / console deletion. Default true as a safety net; override to false only deliberately."
  type        = bool
  default     = true
}

variable "rds_skip_final_snapshot" {
  description = "Whether to skip a final snapshot on destroy. Default false (always snapshot) as a safety net."
  type        = bool
  default     = false
}

# --- RDS storage / monitoring detail (PR #32 review) ---
# All default to AWS's own "not set"/disabled behavior — genuinely optional
# unless the live dev instance actually uses them; set to match before
# import if it does. See terraform/modules/rds/variables.tf for the full
# rationale on each.

variable "rds_shared_preload_libraries" {
  description = "Full shared_preload_libraries list for the dev parameter group (only used when rds_enable_pg_cron is true) — REPLACES the live value, doesn't append. Default [\"pg_cron\"]; list the live instance's complete preload set here if it has others."
  type        = list(string)
  default     = ["pg_cron"]
}

variable "rds_iops" {
  description = "Provisioned IOPS for the dev instance. Null (default) lets AWS use storage_type's default."
  type        = number
  default     = null
}

variable "rds_storage_throughput" {
  description = "gp3 storage throughput (MiBps) for the dev instance. Null (default) lets AWS use gp3's baseline."
  type        = number
  default     = null
}

variable "rds_kms_key_id" {
  description = "KMS key ARN/ID for the dev instance's storage encryption. Null (default) uses the account default key."
  type        = string
  default     = null
}

variable "rds_max_allocated_storage" {
  description = "Storage autoscaling ceiling (GiB) for the dev instance. 0 (default) matches AWS's own \"disabled\" default."
  type        = number
  default     = 0
}

variable "rds_ca_cert_identifier" {
  description = "CA certificate identifier for the dev instance. Null (default) lets AWS use its own default CA."
  type        = string
  default     = null
}

variable "rds_copy_tags_to_snapshot" {
  description = "Whether dev instance snapshots inherit its tags. Default true."
  type        = bool
  default     = true
}

variable "rds_network_type" {
  description = "Network type (\"IPV4\" or \"DUAL\") for the dev instance. Null (default) lets AWS use IPV4."
  type        = string
  default     = null
}

variable "rds_monitoring_interval" {
  description = "Enhanced monitoring interval (seconds) for the dev instance. 0 (default) disables it."
  type        = number
  default     = 0
}

variable "rds_monitoring_role_arn" {
  description = "IAM role ARN for enhanced monitoring on the dev instance. Required by AWS only when rds_monitoring_interval > 0."
  type        = string
  default     = null
}

variable "rds_performance_insights_enabled" {
  description = "Whether Performance Insights is enabled on the dev instance. Default false (AWS default)."
  type        = bool
  default     = false
}

variable "rds_performance_insights_kms_key_id" {
  description = "KMS key for Performance Insights on the dev instance, when enabled. Null (default) uses the account default key."
  type        = string
  default     = null
}

variable "rds_performance_insights_retention_period" {
  description = "Performance Insights retention (days) on the dev instance, when enabled. Default 7 (AWS's own default)."
  type        = number
  default     = 7
}

variable "rds_enabled_cloudwatch_logs_exports" {
  description = "Log types the dev instance exports to CloudWatch Logs. Empty (default) matches AWS's own default of no log exports."
  type        = list(string)
  default     = []
}
