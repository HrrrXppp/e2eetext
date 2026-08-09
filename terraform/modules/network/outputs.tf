output "vpc_id" {
  description = "VPC ID (the account default VPC unless var.vpc_id overrides it)."
  value       = local.vpc_id
}

output "subnet_ids" {
  description = "Subnet IDs in the selected VPC (all of them unless var.subnet_ids overrides)."
  value       = local.subnet_ids
}
