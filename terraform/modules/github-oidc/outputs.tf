output "oidc_provider_arn" {
  description = "ARN of the GitHub Actions OIDC identity provider."
  value       = aws_iam_openid_connect_provider.github.arn
}

output "role_arn" {
  description = "ARN of the IAM role assumed by GitHub Actions (set as the AWS_ROLE_ARN secret)."
  value       = aws_iam_role.github_actions_ecr_push.arn
}

output "policy_arn" {
  description = "ARN of the ECR push permissions policy."
  value       = aws_iam_policy.ecr_push.arn
}
