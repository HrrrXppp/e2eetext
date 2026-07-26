# Phase 1 of #27: account-wide resources with no dev/prod distinction —
# the ECR repos and the GitHub Actions OIDC role. See README.md's
# "Infrastructure as code (Terraform)" section for the required one-time
# `terraform import` bootstrap before the first `terraform apply` here.

provider "aws" {
  region = var.aws_region
}

module "ecr" {
  source = "../../modules/ecr"

  # Matches maintenance/ecr/create-repos.sh exactly: one shared pair, used
  # for both dev and prod (distinguished only by IMAGE_TAG at deploy time).
  repository_names = ["e2eetext-server", "e2eetext-client"]
}

module "github_oidc" {
  source = "../../modules/github-oidc"

  github_org  = var.github_org
  github_repo = var.github_repo

  role_name   = var.github_actions_role_name
  policy_name = var.github_actions_policy_name

  ecr_repository_arns = values(module.ecr.repository_arns)
}
