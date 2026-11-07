# The EC2 instance running the server container. The client is served from
# S3 and CloudFront. CloudFront also connects to this instance on port 8081
# for the API, WebSocket, /sitemap.xml and /robots.txt. A dedicated security
# group allows that port only from the CloudFront origin-facing prefix list,
# so the rule's size does not have to fit in the instance's existing group.
#
# A dev instance already exists (confirmed by the repo owner on #27) — this
# module's job is to describe it precisely enough for `terraform import`,
# not to invent new sizing/AMI defaults. ami_id, instance_type, subnet_id
# etc. have no defaults on purpose: guessing a plausible-sounding default
# risks a silent mismatch against the live instance's actual attributes,
# which forces a replace on `apply` (ami/subnet_id changes are
# ForceNew) — exactly the class of mistake Phase 1 hit with the OIDC
# role name/thumbprint. See README's import bootstrap section.

locals {
  # When the caller passes pre-existing security group IDs (the live dev
  # instance runs with hand-made SGs — "launch-wizard-2" + "ec2-rds-1" —
  # not one this module created), don't create aws_security_group.this:
  # attaching a module-made SG in their place would change the live firewall
  # (and ec2-rds-1's own description warns that detaching it can cut RDS
  # connectivity). aws_security_group.from_cloudfront is attached in
  # addition to those groups, not instead of them.
  create_security_group = length(var.security_group_ids) == 0
  security_group_ids    = local.create_security_group ? [aws_security_group.this[0].id] : var.security_group_ids
}

resource "aws_security_group" "this" {
  count = local.create_security_group ? 1 : 0

  name        = var.security_group_name
  description = "e2eetext EC2 instance: SSH (admin only) + 8080/8081 from the ALB only"
  vpc_id      = var.vpc_id

  dynamic "ingress" {
    for_each = length(var.admin_ssh_cidr_blocks) > 0 ? [1] : []
    content {
      description = "SSH from admin CIDR blocks"
      from_port   = 22
      to_port     = 22
      protocol    = "tcp"
      cidr_blocks = var.admin_ssh_cidr_blocks
    }
  }

  # Port 8081 is on aws_security_group.from_cloudfront, not here. 80/443 stay
  # closed: CloudFront is the public HTTPS endpoint.
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = var.tags
}

# Separate group so the CloudFront prefix list (its entries count toward
# the rules-per-group quota) is not added to launch-wizard-2 or the
# module-created group, which already have their own rules.
#
# The plural data source only calls DescribeManagedPrefixLists, which the
# plan role's ec2:Describe* covers. The singular data source also calls
# GetManagedPrefixListEntries, and that action is not in the plan policy.
data "aws_ec2_managed_prefix_lists" "cloudfront_origin" {
  filter {
    name   = "prefix-list-name"
    values = ["com.amazonaws.global.cloudfront.origin-facing"]
  }
}

resource "aws_security_group" "from_cloudfront" {
  name        = "${var.name != "" ? var.name : var.security_group_name}-cf-origin"
  description = "TCP 8081 from CloudFront origin-facing addresses only"
  vpc_id      = var.vpc_id

  ingress {
    description     = "server from CloudFront"
    from_port       = 8081
    to_port         = 8081
    protocol        = "tcp"
    prefix_list_ids = [one(data.aws_ec2_managed_prefix_lists.cloudfront_origin.ids)]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = var.tags
}

data "aws_iam_policy_document" "assume_role" {
  count = var.create_iam_instance_profile ? 1 : 0

  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["ec2.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "this" {
  count = var.create_iam_instance_profile ? 1 : 0

  name               = var.iam_role_name
  assume_role_policy = data.aws_iam_policy_document.assume_role[0].json
  tags               = var.tags
}

# README's "IAM role: attach AmazonEC2ContainerRegistryReadOnly (so EC2 can
# pull from ECR)" — with the profile attached, maintenance/ec2/ecr-login.sh's
# `aws ecr get-login-password` works without any static credentials on the
# box (the AWS CLI picks up the instance profile automatically via its
# default credential chain). NOTE: despite the README, the live dev
# instance runs with NO instance profile at all (confirmed against AWS,
# 2026-07-26 — the account has zero instance profiles), so ECR pulls there
# must rely on credentials configured on the box. Callers describing such
# an instance set var.create_iam_instance_profile = false; flipping it to
# true later (an in-place instance update) is the migration path to the
# README's documented setup.
resource "aws_iam_role_policy_attachment" "ecr_read_only" {
  count = var.create_iam_instance_profile ? 1 : 0

  role       = aws_iam_role.this[0].name
  policy_arn = "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly"
}

# #37: each server node publishes its own WebSocketConnections count to
# CloudWatch with the instance role; no static credentials on the box.
data "aws_iam_policy_document" "put_metrics" {
  count = var.create_iam_instance_profile && var.metrics_namespace != "" ? 1 : 0

  statement {
    effect    = "Allow"
    actions   = ["cloudwatch:PutMetricData"]
    resources = ["*"]

    condition {
      test     = "StringEquals"
      variable = "cloudwatch:namespace"
      values   = [var.metrics_namespace]
    }
  }
}

resource "aws_iam_role_policy" "put_metrics" {
  count = var.create_iam_instance_profile && var.metrics_namespace != "" ? 1 : 0

  name   = "put-metrics"
  role   = aws_iam_role.this[0].id
  policy = data.aws_iam_policy_document.put_metrics[0].json
}

resource "aws_iam_instance_profile" "this" {
  count = var.create_iam_instance_profile ? 1 : 0

  name = var.iam_instance_profile_name
  role = aws_iam_role.this[0].name
  tags = var.tags
}

resource "aws_instance" "this" {
  ami                    = var.ami_id
  instance_type          = var.instance_type
  subnet_id              = var.subnet_id
  vpc_security_group_ids = concat(local.security_group_ids, [aws_security_group.from_cloudfront.id])
  iam_instance_profile   = var.create_iam_instance_profile ? aws_iam_instance_profile.this[0].name : null
  key_name               = var.key_name != "" ? var.key_name : null

  # Greenfield only in practice: lifecycle ignore_changes below keeps
  # brownfield dig from replacing when this is later enabled. Live dig is
  # configured by aws_ssm_association.boot_deploy instead.
  user_data = local.boot_deploy_enabled ? local.boot_deploy_user_data : null

  # Name tag only when var.name is non-empty — the live, hand-made dev
  # instance carries no tags at all, and forcing a Name tag on it would
  # break the zero-diff import bar. Adding one later is a deliberate
  # in-place change (set var.name), not something import should smuggle in.
  tags = merge(var.tags, var.name != "" ? { Name = var.name } : {})

  lifecycle {
    # user_data / AMI drift on an already-provisioned, hand-configured
    # instance is expected (setup-docker.sh and deploy.sh were run manually
    # over SSH, not via user_data) — don't let Terraform propose replacing a
    # live instance over fields nothing here ever sets. Boot-unit updates for
    # brownfield go through SSM (boot_deploy.tf), not user_data churn.
    ignore_changes = [user_data, user_data_base64]
  }
}
