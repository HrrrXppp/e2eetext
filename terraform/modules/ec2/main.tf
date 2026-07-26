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

resource "aws_security_group" "this" {
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
  name               = var.iam_role_name
  assume_role_policy = data.aws_iam_policy_document.assume_role.json
  tags               = var.tags
}

# README's "IAM role: attach AmazonEC2ContainerRegistryReadOnly (so EC2 can
# pull from ECR)" — this is what already lets maintenance/ec2/ecr-login.sh's
# `aws ecr get-login-password` work without any static credentials on the
# box (the AWS CLI picks up the instance profile automatically via its
# default credential chain). Codifying the existing role/policy here is not
# a functional change to ecr-login.sh or deploy.sh — both already assume
# this role is in place — so neither script needs to change for Phase 2.
resource "aws_iam_role_policy_attachment" "ecr_read_only" {
  role       = aws_iam_role.this.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly"
}

resource "aws_iam_instance_profile" "this" {
  name = var.iam_instance_profile_name
  role = aws_iam_role.this.name
  tags = var.tags
}

resource "aws_instance" "this" {
  ami                    = var.ami_id
  instance_type          = var.instance_type
  subnet_id              = var.subnet_id
  vpc_security_group_ids = [aws_security_group.this.id]
  iam_instance_profile   = aws_iam_instance_profile.this.name
  key_name               = var.key_name != "" ? var.key_name : null

  tags = merge(var.tags, {
    Name = var.name
  })

  lifecycle {
    # user_data / AMI drift on an already-provisioned, hand-configured
    # instance is expected (setup-docker.sh and deploy.sh were run manually
    # over SSH, not via user_data) — don't let Terraform propose replacing a
    # live instance over fields nothing here ever sets.
    ignore_changes = [user_data, user_data_base64]
  }
}
