# Phase 2 of #27: the EC2 instance running docker-compose.yml (server +
# client together — deploy.sh only ever references docker-compose.yml, not
# the split-topology docker-compose.frontend.yml/docker-compose.backend.yml,
# so this module provisions a single combined instance, not two).
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
  # not one this module created), don't create a new SG at all: attaching a
  # module-made SG in their place would change the live firewall (and
  # ec2-rds-1's own description warns that detaching it can cut RDS
  # connectivity).
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

  ingress {
    description     = "client (from ALB only)"
    from_port       = 8080
    to_port         = 8080
    protocol        = "tcp"
    security_groups = [var.alb_security_group_id]
  }

  ingress {
    description     = "server (from ALB only)"
    from_port       = 8081
    to_port         = 8081
    protocol        = "tcp"
    security_groups = [var.alb_security_group_id]
  }

  # No 80/443 here by design — the README explicitly calls out that the ALB
  # handles public HTTP(S), not the instance.
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
  vpc_security_group_ids = local.security_group_ids
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
