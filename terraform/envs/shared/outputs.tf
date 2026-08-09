output "ecr_repository_urls" {
  description = "Map of repository name -> repository URL."
  value       = module.ecr.repository_urls
}

output "github_actions_role_arn" {
  description = "ARN to set as the AWS_ROLE_ARN GitHub secret used by build-images.yml."
  value       = module.github_oidc.role_arn
}
