variable "vpc_id" {
  description = <<-EOT
    Existing VPC ID to use. Leave empty (default) to look up the account's
    default VPC — the expected case for this single-EC2/ALB deployment,
    which has no evidence of a custom VPC anywhere in this repo's scripts
    or docs. Only set this if a given environment's ALB/EC2 actually live
    in a non-default VPC.
  EOT
  type        = string
  default     = ""
}

variable "subnet_ids" {
  description = <<-EOT
    Existing subnet IDs to use for the ALB and EC2 instance. Leave empty
    (default) to use every subnet in the selected VPC (all default-VPC
    subnets are public, which matches create-alb.example.sh's expectation
    of public subnets for the ALB). Set explicitly if only a subset should
    be used, or if the VPC has private subnets that must be excluded.
  EOT
  type        = list(string)
  default     = []
}
