output "role_arn" {
  description = "ARN of the terraform-plan IAM role (AWS_TERRAFORM_PLAN_ROLE_ARN)."
  value       = aws_iam_role.this.arn
}

output "role_name" {
  description = "Name of the terraform-plan IAM role."
  value       = aws_iam_role.this.name
}
