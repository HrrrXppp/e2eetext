variable "name" {
  description = "Name tag for the EC2 instance (e.g. \"e2eetext-dev\")."
  type        = string
}

variable "ami_id" {
  description = <<-EOT
    AMI ID of the live instance. No default: guessing a "latest Amazon
    Linux 2023" AMI here would drift from whatever the real instance
    actually runs and force a replace on `apply`. Get the real value with
    `aws ec2 describe-instances --instance-ids <id> --query
    'Reservations[0].Instances[0].ImageId'` before importing.
  EOT
  type        = string
}

variable "instance_type" {
  description = "Instance type of the live instance (e.g. t3.small per the README's minimum). Must match exactly for a zero-diff import."
  type        = string
}

variable "subnet_id" {
  description = "Subnet ID the instance is launched in (module.network's output, or the specific real subnet ID for import)."
  type        = string
}

variable "vpc_id" {
  description = "VPC ID for the instance's security group (module.network's output)."
  type        = string
}

variable "alb_security_group_id" {
  description = "Security group ID of the ALB — the only source allowed to reach ports 8080/8081 on this instance."
  type        = string
}

variable "admin_ssh_cidr_blocks" {
  description = "CIDR blocks allowed to SSH (port 22) to the instance. Leave empty to omit the SSH ingress rule entirely (e.g. if access is via SSM Session Manager instead)."
  type        = list(string)
  default     = []
}

variable "key_name" {
  description = "EC2 key pair name for SSH access. Leave empty if none is attached (e.g. SSM-only access)."
  type        = string
  default     = ""
}

variable "security_group_name" {
  description = "Name of the instance's security group. Must match the live security group's name for a zero-diff import."
  type        = string
}

variable "iam_role_name" {
  description = "Name of the IAM role backing the instance profile. Must match the live role's name for a zero-diff import."
  type        = string
}

variable "iam_instance_profile_name" {
  description = "Name of the IAM instance profile attached to the instance. Must match the live instance profile's name for a zero-diff import."
  type        = string
}

variable "tags" {
  description = "Tags applied to the instance, security group, and IAM role/instance profile."
  type        = map(string)
  default     = {}
}
