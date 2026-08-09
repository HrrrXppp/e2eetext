variable "aws_region" {
  description = "AWS region for prod's networking/ALB/EC2 resources."
  type        = string
  default     = "us-east-2"
}

variable "name_prefix" {
  description = "Base name for the ALB, target groups, and EC2 instance (e.g. \"e2eetext-prod\")."
  type        = string
  default     = "e2eetext-prod"
}

# --- Networking ---
# No defaults: leaving these empty makes module.network fall back to the
# account's default VPC/subnets. Confirm whether prod actually lives there
# before importing or applying (see main.tf's header comment).

variable "vpc_id" {
  description = "VPC ID prod runs in, if a live stack exists. Leave empty to use the account's default VPC."
  type        = string
  default     = ""
}

variable "subnet_ids" {
  description = "Subnet IDs to consider for placement. Leave empty to use every subnet in the selected VPC."
  type        = list(string)
  default     = []
}

variable "alb_subnet_ids" {
  description = "Public subnet IDs for the ALB specifically. Leave empty to reuse module.network's subnet_ids."
  type        = list(string)
  default     = []
}

# --- EC2 ---
# ami_id and instance_type intentionally have no default — if importing a
# live instance, guessing here risks a forced replace; if applying fresh,
# these should be a deliberate, reviewed choice for a production instance.

variable "ami_id" {
  description = "AMI ID for the prod EC2 instance (real value if importing; a deliberately chosen one if applying fresh)."
  type        = string
}

variable "instance_type" {
  description = "Instance type for the prod EC2 instance (e.g. t3.small or larger, per the README's minimum)."
  type        = string
}

variable "ec2_subnet_id" {
  description = "Subnet ID the prod EC2 instance launches in. Leave empty to use the first subnet from module.network."
  type        = string
  default     = ""
}

variable "alb_security_group_id" {
  description = "Security group ID of the prod ALB (pre-existing, not created by Terraform — see modules/alb)."
  type        = string
}

variable "admin_ssh_cidr_blocks" {
  description = "CIDR blocks allowed to SSH to the prod instance. Leave empty to omit the SSH ingress rule."
  type        = list(string)
  default     = []
}

variable "key_name" {
  description = "EC2 key pair name for the prod instance, if any."
  type        = string
  default     = ""
}

variable "ec2_security_group_ids" {
  description = "Pre-existing security group IDs for the prod instance (skips creating one). Leave empty to have the ec2 module create its own SG. If a live prod instance exists with hand-made SGs (as dev turned out to have), set their IDs here for a zero-diff import."
  type        = list(string)
  default     = []
}

variable "create_ec2_iam_instance_profile" {
  description = "Whether to create/attach the ECR-read IAM role + instance profile. Default true (the README's documented setup) — set false if a live prod instance turns out to run without one, as dev does."
  type        = bool
  default     = true
}

variable "ec2_security_group_name" {
  description = "Name of the prod instance's security group. Must match the live security group's name for a zero-diff import."
  type        = string
  default     = "e2eetext-prod-ec2"
}

variable "ec2_iam_role_name" {
  description = "Name of the prod instance's IAM role. Must match the live role's name for a zero-diff import."
  type        = string
  default     = "e2eetext-prod-ec2-role"
}

variable "ec2_iam_instance_profile_name" {
  description = "Name of the prod instance's IAM instance profile. Must match the live instance profile's name for a zero-diff import."
  type        = string
  default     = "e2eetext-prod-ec2-profile"
}

# --- ALB ---
# Overrides default to the create-alb.example.sh-shaped topology (the
# modules' defaults). If a live prod ALB exists, audit it first and set
# these to its real names/values — the dev audit (2026-07-26) found the
# live stack diverged on every one of them, and the TG/ALB names are
# ForceNew (a mismatch is a replace, i.e. downtime), so don't assume.

variable "alb_name" {
  description = "Name of the prod ALB. Leave empty to use name_prefix. ForceNew — must match a live ALB's real name exactly for a zero-diff import."
  type        = string
  default     = ""
}

variable "server_target_group_name" {
  description = "Name of the server (8081) target group. Leave empty to derive \"<alb name>-server\". ForceNew."
  type        = string
  default     = ""
}

variable "client_target_group_name" {
  description = "Name of the client (8080) target group. Leave empty to derive \"<alb name>-client\". ForceNew."
  type        = string
  default     = ""
}

variable "alb_idle_timeout" {
  description = "ALB idle timeout in seconds. Default 60 to match live-dev / envs/dev (AWS default; create-alb.example.sh's 3600 was never applied on dev)."
  type        = number
  default     = 60
}

variable "https_default_action" {
  description = "HTTPS listener default action. Default \"fixed-404\" to match live-dev / envs/dev (SPA traffic is explicit path rules in additional_https_rules, not a catch-all forward)."
  type        = string
  default     = "fixed-404"
}

variable "api_rule_priority" {
  description = "Priority of the /api listener rule. Default 100 to match live-dev / envs/dev (priorities 10–14 are the client SPA rules)."
  type        = number
  default     = 100
}

variable "api_path_patterns" {
  description = "Path patterns for the /api listener rule. Default [\"/api/*\"] to match live-dev / envs/dev."
  type        = list(string)
  default     = ["/api/*"]
}

variable "create_health_rule" {
  description = "Whether to create the /health -> server listener rule. Default false to match live-dev / envs/dev (TG health checks hit the instance directly)."
  type        = bool
  default     = false
}

variable "additional_https_rules" {
  description = <<-EOT
    Extra HTTPS listener rules (path → client or server TG). Default matches
    live-dev / envs/dev SPA routes (priorities 10–14) with HTTPS default
    action fixed-404. Override to [] only if prod intentionally uses
    forward-client as the HTTPS default instead.
  EOT
  type = list(object({
    priority      = number
    path_patterns = list(string)
    target        = string
  }))
  default = [
    { priority = 10, path_patterns = ["/"], target = "client" },
    { priority = 11, path_patterns = ["/assets/*"], target = "client" },
    { priority = 12, path_patterns = ["/oauth/*"], target = "client" },
    { priority = 13, path_patterns = ["/chats"], target = "client" },
    { priority = 14, path_patterns = ["/instance.json"], target = "client" },
  ]
}


variable "health_check_healthy_threshold" {
  description = "TG healthy threshold. Default 5 to match live-dev / envs/dev (not the provider default of 3)."
  type        = number
  default     = 5
}

variable "health_check_unhealthy_threshold" {
  description = "TG unhealthy threshold. Default 2 to match live-dev / envs/dev (not the provider default of 3)."
  type        = number
  default     = 2
}

variable "server_health_check_port" {
  description = "Server TG health-check port. Default \"8081\" to match live-dev / envs/dev."
  type        = string
  default     = "8081"
}

variable "client_health_check_port" {
  description = "Client TG health-check port. Default \"traffic-port\" to match live-dev / envs/dev."
  type        = string
  default     = "traffic-port"
}

variable "client_health_check_path" {
  description = "Client TG health-check path. Default \"/\" to match live-dev / envs/dev (not \"/health\")."
  type        = string
  default     = "/"
}

variable "acm_certificate_arn" {
  description = "ACM certificate ARN for the prod ALB's HTTPS listener (production domain)."
  type        = string
}

variable "tags" {
  description = "Tags applied to all prod resources."
  type        = map(string)
  default = {
    Environment = "prod"
    ManagedBy   = "terraform"
  }
}

# --- RDS (Phase 3 of #27) ---
# Live status not yet confirmed (see main.tf's rds module header) — audit
# first, then either import real values or apply fresh with deliberate,
# reviewed choices. Required variables below have no default for the same
# reason ami_id/instance_type/alb_security_group_id/acm_certificate_arn
# don't: a wrong guess for an import target risks a forced replace, and for
# a fresh apply these should be deliberate choices, not defaults.

variable "rds_identifier" {
  description = "DB instance identifier (real value if importing; a deliberately chosen one, e.g. \"e2eetext-prod\", if applying fresh)."
  type        = string
}

variable "rds_engine_version" {
  description = "Engine version, e.g. \"16.4\" (real value if importing; a deliberately chosen supported PostgreSQL 16.x minor version if applying fresh)."
  type        = string
}

variable "rds_instance_class" {
  description = "Instance class, e.g. \"db.t3.micro\" (real value if importing; sized for expected prod load if applying fresh)."
  type        = string
}

variable "rds_allocated_storage" {
  description = "Allocated storage (GiB)."
  type        = number
}

variable "rds_storage_type" {
  description = "Storage type, e.g. \"gp3\"."
  type        = string
}

variable "rds_storage_encrypted" {
  description = "Whether storage is encrypted at rest. ForceNew if it ever needs to change."
  type        = bool
}

variable "rds_db_name" {
  description = <<-EOT
    Initial CreateDBInstance DBName. ForceNew — leave null (default) when
    the live instance has no initial DBName (AWS returns null). Set only
    when the instance was actually created with that name. For the app's
    PostgreSQL database, see rds_app_database_name.
  EOT
  type        = string
  default     = null
  nullable    = true
}

variable "rds_app_database_name" {
  description = "PostgreSQL database the app connects to (DATABASE_URL path). Not ForceNew. Default \"messenger\" until prod's live DB name is audited."
  type        = string
  default     = "messenger"
}

variable "rds_username" {
  description = "Master username. No default: README's AWS walkthrough treats this as chosen at creation time."
  type        = string
}

variable "rds_password" {
  description = "Master password. Never commit this — pass via TF_VAR_rds_password or -var, sourced from Secrets Manager/SSM, not terraform.tfvars."
  type        = string
  sensitive   = true
}

variable "rds_subnet_ids" {
  description = <<-EOT
    Subnet IDs for the DB subnet group. No default (PR #32 review): this
    used to fall back to module.network's subnets when unset, which risks
    silently landing RDS in public default-VPC subnets — RDS is commonly
    placed in private subnets instead. Confirm the real (or intended)
    subnet-group members before importing or applying.
  EOT
  type        = list(string)
}

variable "rds_subnet_group_name" {
  description = "Name of the DB subnet group (real live name if importing; a deliberately chosen one if applying fresh)."
  type        = string
}

variable "rds_create_subnet_group" {
  description = <<-EOT
    Whether the rds module creates/manages the DB subnet group. Default
    true; set false when rds_subnet_group_name is a pre-existing group this
    module shouldn't try to create — including the AWS-implicit "default"
    DB subnet group every default VPC has, which the provider refuses to
    create (name "default" is reserved). See envs/dev's identical variable
    for the live failure this was confirmed against.
  EOT
  type        = bool
  default     = true
}

variable "rds_security_group_ids" {
  description = "Pre-existing security group IDs for the prod RDS instance (skips creating one). Leave empty to have the rds module create its own SG."
  type        = list(string)
  default     = []
}

variable "rds_security_group_name" {
  description = "Name for the prod RDS security group, if the rds module creates one."
  type        = string
  default     = "e2eetext-prod-rds"
}

variable "rds_ingress_security_group_id" {
  description = "Security group ID allowed to reach 5432 on the created RDS SG. Leave empty to fall back to module.ec2's own SG (a real value here by default, since prod's ec2 module creates its own SG unless ec2_security_group_ids is overridden)."
  type        = string
  default     = ""
}

variable "rds_create_parameter_group" {
  description = "Whether the rds module creates/manages the DB parameter group. Default true so Phase 3 folds in #20's pg_cron settings."
  type        = bool
  default     = true
}

variable "rds_parameter_group_name" {
  description = "Name of the prod DB parameter group."
  type        = string
  default     = "e2eetext-prod-rds-pgcron"
}

variable "rds_parameter_group_family" {
  description = "Parameter group family. Default \"postgres16\" per README's fixed tech-stack version — must match rds_engine_version's major version exactly."
  type        = string
  default     = "postgres16"
}

variable "rds_enable_pg_cron" {
  description = "Whether to set shared_preload_libraries=pg_cron + cron.database_name on the parameter group, per #20's plan. Default true."
  type        = bool
  default     = true
}

variable "rds_cron_database_name" {
  description = "Value for cron.database_name. Leave empty (default) to reuse rds_db_name."
  type        = string
  default     = ""
}

variable "rds_multi_az" {
  description = "Whether the instance is Multi-AZ. No default — a real availability decision for production, not a guess."
  type        = bool
}

variable "rds_publicly_accessible" {
  description = "Whether the prod RDS instance has a public IP. Default false — README's AWS walkthrough is explicit that PostgreSQL must not be exposed to the public internet."
  type        = bool
  default     = false
}

variable "rds_backup_retention_period" {
  description = "Automated backup retention in days (0 = disabled)."
  type        = number
}

variable "rds_backup_window" {
  description = "Preferred backup window (UTC), e.g. \"03:00-04:00\"."
  type        = string
}

variable "rds_maintenance_window" {
  description = "Preferred maintenance window (UTC), e.g. \"sun:04:30-sun:05:30\"."
  type        = string
}

variable "rds_deletion_protection" {
  description = "Whether the prod RDS instance is protected from `terraform destroy` / console deletion. Default true as a safety net."
  type        = bool
  default     = true
}

variable "rds_skip_final_snapshot" {
  description = "Whether to skip a final snapshot on destroy. Default false (always snapshot) as a safety net for production data."
  type        = bool
  default     = false
}

# --- RDS storage / monitoring detail (PR #32 review) ---
# All default to AWS's own "not set"/disabled behavior — genuinely optional
# unless the live prod instance actually uses them; set to match before
# import if it does. See terraform/modules/rds/variables.tf for the full
# rationale on each.

variable "rds_shared_preload_libraries" {
  description = "Full shared_preload_libraries list for the prod parameter group (only used when rds_enable_pg_cron is true) — REPLACES the live value, doesn't append. Default [\"pg_cron\"]; list the live instance's complete preload set here if it has others."
  type        = list(string)
  default     = ["pg_cron"]
}

variable "rds_iops" {
  description = "Provisioned IOPS for the prod instance. Null (default) lets AWS use storage_type's default."
  type        = number
  default     = null
}

variable "rds_storage_throughput" {
  description = "gp3 storage throughput (MiBps) for the prod instance. Null (default) lets AWS use gp3's baseline."
  type        = number
  default     = null
}

variable "rds_kms_key_id" {
  description = "KMS key ARN/ID for the prod instance's storage encryption. Null (default) uses the account default key."
  type        = string
  default     = null
}

variable "rds_max_allocated_storage" {
  description = "Storage autoscaling ceiling (GiB) for the prod instance. 0 (default) matches AWS's own \"disabled\" default."
  type        = number
  default     = 0
}

variable "rds_ca_cert_identifier" {
  description = "CA certificate identifier for the prod instance. Null (default) lets AWS use its own default CA."
  type        = string
  default     = null
}

variable "rds_copy_tags_to_snapshot" {
  description = "Whether prod instance snapshots inherit its tags. Default true."
  type        = bool
  default     = true
}

variable "rds_network_type" {
  description = "Network type (\"IPV4\" or \"DUAL\") for the prod instance. Null (default) lets AWS use IPV4."
  type        = string
  default     = null
}

variable "rds_monitoring_interval" {
  description = "Enhanced monitoring interval (seconds) for the prod instance. 0 (default) disables it."
  type        = number
  default     = 0
}

variable "rds_monitoring_role_arn" {
  description = "IAM role ARN for enhanced monitoring on the prod instance. Required by AWS only when rds_monitoring_interval > 0."
  type        = string
  default     = null
}

variable "rds_performance_insights_enabled" {
  description = "Whether Performance Insights is enabled on the prod instance. Default false (AWS default)."
  type        = bool
  default     = false
}

variable "rds_performance_insights_kms_key_id" {
  description = "KMS key for Performance Insights on the prod instance, when enabled. Null (default) uses the account default key."
  type        = string
  default     = null
}

variable "rds_performance_insights_retention_period" {
  description = "Performance Insights retention (days) on the prod instance, when enabled. Default 7 (AWS's own default)."
  type        = number
  default     = 7
}

variable "rds_enabled_cloudwatch_logs_exports" {
  description = "Log types the prod instance exports to CloudWatch Logs. Empty (default) matches AWS's own default of no log exports."
  type        = list(string)
  default     = []
}
