# Phase 2 of #27: networking for one environment's ALB + EC2 instance.
#
# This is a single small deployment (one EC2 instance behind one ALB per
# environment), not a fresh multi-tier AWS account build-out. There's no
# evidence anywhere in this repo (scripts, README) of a custom VPC ever
# being created — maintenance/alb/create-alb.example.sh takes VPC_ID and
# SUBNET_IDS as pre-existing inputs, and nothing under maintenance/ creates
# a VPC. The most likely reality is that both dev and prod live in the
# account's default VPC and its default subnets.
#
# Because of that, this module is deliberately data-source-only: it never
# creates, modifies, or destroys a VPC/subnet. That makes it safe regardless
# of whether the caller's environment already has live resources sitting in
# that VPC (confirmed true for dev — see README's Terraform import bootstrap
# section) — a data source can't collide with or destroy anything.
#
# If a given environment turns out to use a non-default VPC, pass its ID
# explicitly via var.vpc_id (and var.subnet_ids if not all of that VPC's
# subnets should be considered) rather than changing this module to create
# one.

data "aws_vpc" "default" {
  count   = var.vpc_id == "" ? 1 : 0
  default = true
}

locals {
  vpc_id = var.vpc_id != "" ? var.vpc_id : data.aws_vpc.default[0].id
}

data "aws_subnets" "selected" {
  count = length(var.subnet_ids) == 0 ? 1 : 0

  filter {
    name   = "vpc-id"
    values = [local.vpc_id]
  }
}

locals {
  subnet_ids = length(var.subnet_ids) > 0 ? var.subnet_ids : data.aws_subnets.selected[0].ids
}
