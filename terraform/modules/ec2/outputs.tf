output "instance_id" {
  description = "EC2 instance ID."
  value       = aws_instance.this.id
}

output "public_dns" {
  description = "Public DNS name CloudFront uses as the server origin. Empty when the instance has no public IPv4 address. Changes if the instance is stopped and started without an Elastic IP."
  value       = aws_instance.this.public_dns
}

output "security_group_id" {
  description = "Security group ID created for the instance, or null when pre-existing SGs were passed via var.security_group_ids."
  value       = local.create_security_group ? aws_security_group.this[0].id : null
}

output "iam_role_arn" {
  description = "ARN of the IAM role backing the instance profile, or null when var.create_iam_instance_profile is false."
  value       = var.create_iam_instance_profile ? aws_iam_role.this[0].arn : null
}

output "iam_instance_profile_name" {
  description = "Name of the IAM instance profile attached to the instance, or null when var.create_iam_instance_profile is false."
  value       = var.create_iam_instance_profile ? aws_iam_instance_profile.this[0].name : null
}

output "boot_deploy_association_id" {
  description = "SSM association ID for the boot deploy unit, or null when manage_boot_deploy is false."
  value       = var.manage_boot_deploy ? aws_ssm_association.boot_deploy[0].association_id : null
}
