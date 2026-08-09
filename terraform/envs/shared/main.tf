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

  role_name          = var.github_actions_role_name
  inline_policy_name = var.github_actions_inline_policy_name

  oidc_thumbprint_list = var.oidc_thumbprint_list

  ecr_repository_arns = values(module.ecr.repository_arns)
}

# Read-only role for terraform.yml plan jobs (AWS_TERRAFORM_PLAN_ROLE_ARN).
# Import the live hand-made role + inline policy before the first apply —
# see README "terraform-plan-role".
module "terraform_plan_role" {
  source = "../../modules/terraform-plan-role"

  github_org          = var.github_org
  github_repo         = var.github_repo
  oidc_provider_arn   = module.github_oidc.oidc_provider_arn
  ecr_repository_arns = values(module.ecr.repository_arns)
  role_name           = var.terraform_plan_role_name
  inline_policy_name  = var.terraform_plan_inline_policy_name
  state_bucket        = var.terraform_state_bucket
}
