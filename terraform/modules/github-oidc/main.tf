# OIDC identity provider trusting GitHub Actions' token issuer, used by
# .github/workflows/build-images.yml (aws-actions/configure-aws-credentials
# with role-to-assume: ${{ secrets.AWS_ROLE_ARN }}).
#
# thumbprint_list is explicitly set (from var.oidc_thumbprint_list) rather
# than omitted. The hashicorp/aws provider version this module is locked to
# (5.100.0, satisfying the "~> 5.0" constraint in versions.tf — verified via
# `terraform providers schema`) does expose this argument as Optional +
# Computed, meaning AWS can auto-validate GitHub's well-known OIDC issuer
# chain and Terraform would leave an already-recorded value alone if the
# argument were simply omitted. That makes dropping it a plausible future
# simplification. It isn't done here because: (a) "~> 5.0" itself allows
# upgrading to other 5.x releases where this behavior may differ, so
# omitting it ties correctness to whichever provider version happens to be
# resolved rather than to an explicit, reviewable value; and (b) this fix's
# job is narrowly to correct the wrong hardcoded value against live state
# (verified with no AWS access, via the CI plan log), not to change how the
# attribute is managed. Revisit once the provider constraint is pinned
# tighter and someone can confirm the omitted-argument behavior against a
# real `terraform plan`.
resource "aws_iam_openid_connect_provider" "github" {
  url             = "https://token.actions.githubusercontent.com"
  client_id_list  = ["sts.amazonaws.com"]
  thumbprint_list = var.oidc_thumbprint_list
}

# Trust policy scoped to the whole repository (repo:OWNER/REPO:*) — not a
# branch or environment, matching the manually-created role. This is
# intentionally broad for the zero-diff Phase 1 import (any ref/workflow in
# the repo, including workflow_dispatch from an arbitrary branch, can assume
# this role): tightening it now would diverge from what's actually live and
# break the "terraform plan shows no changes" import bar. Once import is
# confirmed, consider narrowing the `sub` condition below to specific refs
# (e.g. "repo:${var.github_org}/${var.github_repo}:ref:refs/heads/main") and/or
# fronting it with a GitHub Environment that requires reviewers.
data "aws_iam_policy_document" "trust" {
  statement {
    sid     = "GitHubActionsAssumeRoleWithWebIdentity"
    effect  = "Allow"
    actions = ["sts:AssumeRoleWithWebIdentity"]

    principals {
      type        = "Federated"
      identifiers = [aws_iam_openid_connect_provider.github.arn]
    }

    condition {
      test     = "StringEquals"
      variable = "token.actions.githubusercontent.com:aud"
      values   = ["sts.amazonaws.com"]
    }

    condition {
      test     = "StringLike"
      variable = "token.actions.githubusercontent.com:sub"
      values   = ["repo:${var.github_org}/${var.github_repo}:*"]
    }
  }
}

resource "aws_iam_role" "github_actions_ecr_push" {
  name               = var.role_name
  assume_role_policy = data.aws_iam_policy_document.trust.json
  tags               = var.tags
}

# Permissions policy: ECR auth (account-wide, ECR requires resource "*" for
# GetAuthorizationToken) plus the exact push actions build-and-push.sh needs,
# scoped to just the two repos (see README's "Requires IAM permissions" note).
data "aws_iam_policy_document" "ecr_push" {
  statement {
    sid       = "ECRAuth"
    effect    = "Allow"
    actions   = ["ecr:GetAuthorizationToken"]
    resources = ["*"]
  }

  statement {
    sid    = "ECRPush"
    effect = "Allow"
    actions = [
      "ecr:BatchCheckLayerAvailability",
      "ecr:PutImage",
      "ecr:InitiateLayerUpload",
      "ecr:UploadLayerPart",
      "ecr:CompleteLayerUpload",
    ]
    resources = var.ecr_repository_arns
  }
}

resource "aws_iam_policy" "ecr_push" {
  name   = var.policy_name
  policy = data.aws_iam_policy_document.ecr_push.json
}

resource "aws_iam_role_policy_attachment" "ecr_push" {
  role       = aws_iam_role.github_actions_ecr_push.name
  policy_arn = aws_iam_policy.ecr_push.arn
}
