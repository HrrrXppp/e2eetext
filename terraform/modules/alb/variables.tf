variable "name" {
  description = "Base name for the ALB and its target groups (e.g. \"e2eetext-dev\"). Must match the live ALB's name for a zero-diff import."
  type        = string
}

variable "vpc_id" {
  description = "VPC ID the target groups live in (module.network's output)."
  type        = string
}

variable "subnet_ids" {
  description = "Public subnet IDs for the internet-facing ALB (module.network's output, or an explicit subset)."
  type        = list(string)
}

variable "security_group_ids" {
  description = <<-EOT
    Existing security group ID(s) for the ALB, allowing inbound 80/443 from
    0.0.0.0/0. Not created by this module — create-alb.example.sh treats
    ALB_SG_ID as a pre-existing input, and this module mirrors that. Set to
    the real, already-live security group ID for a zero-diff import.
  EOT
  type        = list(string)
}

variable "instance_id" {
  description = "EC2 instance ID to register in both target groups (module.ec2's output)."
  type        = string
}

variable "acm_certificate_arn" {
  description = "ACM certificate ARN for the HTTPS listener (must be in the same region as the ALB)."
  type        = string
}

variable "tags" {
  description = "Tags applied to the ALB and target groups."
  type        = map(string)
  default     = {}
}
