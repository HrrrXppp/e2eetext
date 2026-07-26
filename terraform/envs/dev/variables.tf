variable "aws_region" {
  description = "AWS region for dev's networking/ALB/EC2 resources."
  type        = string
  default     = "us-east-2"
}

variable "name_prefix" {
  description = "Base name for the ALB, target groups, and EC2 instance (e.g. \"e2eetext-dev\")."
  type        = string
  default     = "e2eetext-dev"
}

# --- Networking ---
# No defaults: leaving these empty makes module.network fall back to the
# account's default VPC/subnets, which is a reasonable guess but must be
# confirmed against what's actually live before importing (see README).

variable "vpc_id" {
  description = "VPC ID the live dev stack runs in. Leave empty to use the account's default VPC."
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
# ami_id and instance_type intentionally have no default — see
# modules/ec2/variables.tf's comment on why guessing here is unsafe for an
# import target.

variable "ami_id" {
  description = "AMI ID of the live dev EC2 instance. Get the real value with `aws ec2 describe-instances` before importing."
  type        = string
}

variable "instance_type" {
  description = "Instance type of the live dev EC2 instance (e.g. t3.small)."
  type        = string
}

variable "ec2_subnet_id" {
  description = "Subnet ID the dev EC2 instance actually launches in. Leave empty to use the first subnet from module.network."
  type        = string
  default     = ""
}

variable "alb_security_group_id" {
  description = "Security group ID of the dev ALB (pre-existing, not created by Terraform — see modules/alb)."
  type        = string
}

variable "admin_ssh_cidr_blocks" {
  description = "CIDR blocks allowed to SSH to the dev instance. Leave empty to omit the SSH ingress rule."
  type        = list(string)
  default     = []
}

variable "key_name" {
  description = "EC2 key pair name for the dev instance, if any."
  type        = string
  default     = ""
}

variable "ec2_security_group_name" {
  description = "Name of the dev instance's security group. Must match the live security group's name for a zero-diff import."
  type        = string
  default     = "e2eetext-dev-ec2"
}

variable "ec2_iam_role_name" {
  description = "Name of the dev instance's IAM role. Must match the live role's name for a zero-diff import."
  type        = string
  default     = "e2eetext-dev-ec2-role"
}

variable "ec2_iam_instance_profile_name" {
  description = "Name of the dev instance's IAM instance profile. Must match the live instance profile's name for a zero-diff import."
  type        = string
  default     = "e2eetext-dev-ec2-profile"
}

# --- ALB ---

variable "acm_certificate_arn" {
  description = "ACM certificate ARN for the dev ALB's HTTPS listener."
  type        = string
}

variable "tags" {
  description = "Tags applied to all dev resources."
  type        = map(string)
  default = {
    Environment = "dev"
    ManagedBy   = "terraform"
  }
}
