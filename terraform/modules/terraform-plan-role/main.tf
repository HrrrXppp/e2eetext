# Read-only IAM role assumed by .github/workflows/terraform.yml (plan only).
# Lives in envs/shared with the GitHub OIDC provider. The live role was
# created by hand with an *inline* policy (not a managed policy) — this
# module matches that shape so import stays clean, then owns policy updates
# (e.g. dig/prod EC2 instance-profile + SSM boot-deploy reads).

data "aws_caller_identity" "current" {}

locals {
  account_id = data.aws_caller_identity.current.account_id
}

data "aws_iam_policy_document" "trust" {
  statement {
    sid     = "GitHubActionsAssumeRoleWithWebIdentity"
    effect  = "Allow"
    actions = ["sts:AssumeRoleWithWebIdentity"]

    principals {
      type        = "Federated"
      identifiers = [var.oidc_provider_arn]
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

resource "aws_iam_role" "this" {
  name               = var.role_name
  assume_role_policy = data.aws_iam_policy_document.trust.json
  tags               = var.tags
}

data "aws_iam_policy_document" "plan" {
  statement {
    sid    = "TerraformStateBucket"
    effect = "Allow"
    actions = [
      "s3:ListBucket",
      "s3:GetBucketVersioning",
      "s3:GetBucketLocation",
    ]
    resources = ["arn:aws:s3:::${var.state_bucket}"]
  }

  statement {
    sid       = "TerraformStateReadOnly"
    effect    = "Allow"
    actions   = ["s3:GetObject"]
    resources = ["arn:aws:s3:::${var.state_bucket}/envs/*/terraform.tfstate"]
  }

  statement {
    sid    = "ECRRead"
    effect = "Allow"
    actions = [
      "ecr:DescribeRepositories",
      "ecr:ListTagsForResource",
      "ecr:GetLifecyclePolicy",
      "ecr:GetRepositoryPolicy",
      "ecr:DescribeImageScanFindings",
    ]
    resources = var.ecr_repository_arns
  }

  statement {
    sid    = "IAMReadManaged"
    effect = "Allow"
    actions = [
      "iam:GetRole",
      "iam:GetRolePolicy",
      "iam:ListRolePolicies",
      "iam:ListAttachedRolePolicies",
      "iam:GetInstanceProfile",
      "iam:ListInstanceProfilesForRole",
      "iam:GetPolicy",
      "iam:GetPolicyVersion",
      "iam:ListPolicyVersions",
      "iam:GetOpenIDConnectProvider",
      "iam:ListOpenIDConnectProviders",
      "iam:ListRoleTags",
      "iam:ListPolicyTags",
      "iam:ListInstanceProfileTags",
    ]
    resources = [
      "arn:aws:iam::${local.account_id}:role/github-actions-ecr-role",
      "arn:aws:iam::${local.account_id}:policy/github-actions-ecr-role",
      "arn:aws:iam::${local.account_id}:policy/github-actions-ecr-push",
      "arn:aws:iam::${local.account_id}:oidc-provider/token.actions.githubusercontent.com",
      # This role (so shared plan can refresh itself after import).
      "arn:aws:iam::${local.account_id}:role/${var.role_name}",
      # dig / prod EC2 instance profiles (boot deploy + ECR pull).
      "arn:aws:iam::${local.account_id}:role/e2eetext-dev-ec2-role",
      "arn:aws:iam::${local.account_id}:role/e2eetext-prod-ec2-role",
      "arn:aws:iam::${local.account_id}:instance-profile/e2eetext-dev-ec2-profile",
      "arn:aws:iam::${local.account_id}:instance-profile/e2eetext-prod-ec2-profile",
      "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly",
      "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore",
    ]
  }

  statement {
    sid    = "IAMListAccount"
    effect = "Allow"
    actions = [
      "iam:ListRoles",
      "iam:ListPolicies",
      "iam:ListInstanceProfiles",
    ]
    resources = ["*"]
  }

  statement {
    sid    = "NetworkAlbEc2AcmRead"
    effect = "Allow"
    actions = [
      "ec2:Describe*",
      "elasticloadbalancing:Describe*",
      "acm:DescribeCertificate",
      "acm:ListCertificates",
      "acm:ListTagsForCertificate",
    ]
    resources = ["*"]
  }

  statement {
    sid    = "RdsRead"
    effect = "Allow"
    actions = [
      "rds:DescribeDBInstances",
      "rds:DescribeDBSubnetGroups",
      "rds:DescribeDBParameterGroups",
      "rds:DescribeDBParameters",
      "rds:ListTagsForResource",
    ]
    resources = ["*"]
  }

  statement {
    sid    = "SsmBootDeployRead"
    effect = "Allow"
    actions = [
      "ssm:DescribeDocument",
      "ssm:DescribeDocumentPermission",
      "ssm:ListTagsForResource",
      "ssm:GetDocument",
      "ssm:DescribeAssociation",
      "ssm:ListAssociations",
      "ssm:DescribeAssociationExecutions",
    ]
    resources = [
      "arn:aws:ssm:*:${local.account_id}:document/e2eetext-boot-deploy",
      "arn:aws:ssm:*:${local.account_id}:document/e2eetext-*-boot-deploy",
      "arn:aws:ssm:*:${local.account_id}:association/*",
    ]
  }
}

resource "aws_iam_role_policy" "this" {
  name   = var.inline_policy_name
  role   = aws_iam_role.this.id
  policy = data.aws_iam_policy_document.plan.json
}
