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
  description = "ALB idle timeout in seconds. Default 3600 per create-alb.example.sh (websockets); check what's actually live before importing."
  type        = number
  default     = 3600
}

variable "https_default_action" {
  description = "HTTPS listener default action: \"forward-client\" (the script's shape, default) or \"fixed-404\" (what live dev has)."
  type        = string
  default     = "forward-client"
}

variable "api_rule_priority" {
  description = "Priority of the /api listener rule (script default 10; live dev uses 100)."
  type        = number
  default     = 10
}

variable "api_path_patterns" {
  description = "Path patterns for the /api listener rule (script default [\"/api*\"]; live dev uses [\"/api/*\"])."
  type        = list(string)
  default     = ["/api*"]
}

variable "create_health_rule" {
  description = "Whether to create the /health -> server listener rule (script default true; live dev has no such rule)."
  type        = bool
  default     = true
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
