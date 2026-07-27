variable "name" {
  description = "Name tag for the EC2 instance (e.g. \"e2eetext-dev\"). Leave empty to set no Name tag — required for a zero-diff import of the live dev instance, which is untagged."
  type        = string
  default     = ""
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

variable "security_group_ids" {
  description = <<-EOT
    Pre-existing security group IDs to attach to the instance instead of
    creating one. Leave empty (default) to have this module create its own
    SG (named var.security_group_name, allowing 8080/8081 from the ALB).
    Set to the instance's real, live security groups when importing an
    instance whose SGs were made by hand — swapping them for a module-made
    SG is an in-place update but a real firewall change (e.g. detaching a
    live ec2-rds-* group can cut RDS connectivity).
  EOT
  type        = list(string)
  default     = []
}

variable "create_iam_instance_profile" {
  description = <<-EOT
    Whether to create the IAM role + instance profile (with
    AmazonEC2ContainerRegistryReadOnly) and attach it to the instance. Set
    false when importing a live instance that runs without any instance
    profile (true of the live dev instance) so the import stays zero-diff;
    flip to true later to migrate to the README's documented ECR-pull
    setup.
  EOT
  type        = bool
  default     = true
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
