output "instance_id" {
  description = "EC2 instance ID — register this in the ALB's target groups."
  value       = aws_instance.this.id
}

output "security_group_id" {
  description = "Security group ID attached to the instance."
  value       = aws_security_group.this.id
}

output "iam_role_arn" {
  description = "ARN of the IAM role backing the instance profile."
  value       = aws_iam_role.this.arn
}

output "iam_instance_profile_name" {
  description = "Name of the IAM instance profile attached to the instance."
  value       = aws_iam_instance_profile.this.name
}
