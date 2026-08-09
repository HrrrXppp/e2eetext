variable "github_org" {
  description = "GitHub organization/user that owns the repository (e.g. HrrrXppp)."
  type        = string
}

variable "github_repo" {
  description = "GitHub repository name (e.g. e2eetext)."
  type        = string
}

variable "oidc_thumbprint_list" {
  description = <<-EOT
    TLS certificate thumbprints for token.actions.githubusercontent.com's certificate
    authority chain. This must match the thumbprint AWS already has recorded for the
    live, manually-created OIDC provider (confirmed via
    `aws iam get-open-id-connect-provider`), or Terraform will plan an in-place update
    that changes the live thumbprint and risks breaking OIDC federation if it doesn't
    match GitHub's actual current certificate chain. The value below is NOT the
    commonly-copied "well-known" GitHub thumbprint from older tutorials/docs
    (6938fd4d98bab03faadb97b34396831e3780aea1) — that value doesn't match what's
    actually live for this provider. Note the real chain can rotate over time (e.g. if
    GitHub changes CAs), so this may need to be re-verified against AWS periodically.
  EOT
  type        = list(string)
  default     = ["ab9d0263244dd0326eb67015705a667e79cfe998"]
}

variable "role_name" {
  description = "Name of the IAM role assumed by GitHub Actions via OIDC. Must match the role already created manually, for a zero-diff import."
  type        = string
  # Matches the live role's actual name (see .github/workflows/build-images.yml's
  # secrets.AWS_ROLE_ARN, which points at this role). A prior default of
  # "github-actions-ecr-push" didn't match, which made Terraform plan to
  # destroy the live role and create a new one under that name instead of
  # recognizing it — that would have broken build-images.yml the moment
  # apply ran, since AWS_ROLE_ARN would still point at the deleted role.
  default = "github-actions-ecr-role"
}

variable "policy_name" {
  description = "Name of the IAM policy granting ECR push access, attached to role_name. Must match the policy already created manually, for a zero-diff import."
  type        = string
  default     = "github-actions-ecr-push"
}

variable "ecr_repository_arns" {
  description = "ARNs of the ECR repositories the role is allowed to push/pull. Scopes the permissions policy to just these repos, matching the manually-created policy."
  type        = list(string)
}

variable "tags" {
  description = "Tags applied to the IAM role."
  type        = map(string)
  default     = {}
}
