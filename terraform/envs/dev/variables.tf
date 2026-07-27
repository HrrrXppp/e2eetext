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
