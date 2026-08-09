output "oidc_provider_arn" {
  description = "ARN of the GitHub Actions OIDC identity provider."
  value       = aws_iam_openid_connect_provider.github.arn
}

output "role_arn" {
  description = "ARN of the IAM role assumed by GitHub Actions (set as the AWS_ROLE_ARN secret)."
  value       = aws_iam_role.github_actions_ecr_push.arn
}

output "inline_policy_name" {
  description = "Name of the inline ECR push policy on the GitHub Actions role."
  value       = aws_iam_role_policy.ecr_push.name
}
