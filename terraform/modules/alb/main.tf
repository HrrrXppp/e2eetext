# Phase 2 of #27: mirrors maintenance/alb/create-alb.example.sh exactly —
# same target group ports/health checks, same listener/rule shape. That
# script is the closest thing to a spec for what's actually live (per the
# repo owner, a dev EC2/ALB/VPC stack already exists), so this module's job
# is to describe those existing resources for `terraform import`, not to
# invent a new topology.
#
# Deliberately does NOT create the ALB's own security group: the script
# treats ALB_SG_ID as a pre-existing input ("allow 80/443 from 0.0.0.0/0")
# rather than something it creates, so this module follows suit and takes
# var.security_group_ids as-is. See README's import bootstrap section for
# how to import the rest of what this module manages.

locals {
  server_target_group_name = var.server_target_group_name != "" ? var.server_target_group_name : "${var.name}-server"
  client_target_group_name = var.client_target_group_name != "" ? var.client_target_group_name : "${var.name}-client"
}

resource "aws_lb" "this" {
  name               = var.name
  internal           = false
  load_balancer_type = "application"
  security_groups    = var.security_group_ids
  subnets            = var.subnet_ids

  # create-alb.example.sh sets 3600 (the module default) for /ws WebSocket
  # support; the live dev ALB runs the AWS default of 60 — see
  # var.idle_timeout.
  idle_timeout = var.idle_timeout

  tags = var.tags
}

# Port 8081, health check /health — matches create-alb.example.sh's
# "server target group" section exactly.
resource "aws_lb_target_group" "server" {
  name        = local.server_target_group_name
  port        = 8081
  protocol    = "HTTP"
  vpc_id      = var.vpc_id
  target_type = "instance"

  health_check {
    path                = "/health"
    interval            = 30
    port                = var.server_health_check_port
    healthy_threshold   = var.health_check_healthy_threshold
    unhealthy_threshold = var.health_check_unhealthy_threshold
  }

  tags = var.tags
}

# Port 8080, health check /health — matches create-alb.example.sh's
# "client target group" section exactly (yes, the client TG also uses
# /health as its health-check path per the script, even though it's the
# default/catch-all target rather than the one serving GET /health).
resource "aws_lb_target_group" "client" {
  name        = local.client_target_group_name
  port        = 8080
  protocol    = "HTTP"
  vpc_id      = var.vpc_id
  target_type = "instance"

  health_check {
    path                = var.client_health_check_path
    interval            = 30
    port                = var.client_health_check_port
    healthy_threshold   = var.health_check_healthy_threshold
    unhealthy_threshold = var.health_check_unhealthy_threshold
  }

  tags = var.tags
}

resource "aws_lb_target_group_attachment" "server" {
  target_group_arn = aws_lb_target_group.server.arn
  target_id        = var.instance_id
  port             = 8081
}

resource "aws_lb_target_group_attachment" "client" {
  target_group_arn = aws_lb_target_group.client.arn
  target_id        = var.instance_id
  port             = 8080
}

# Default action: forward to the client TG ("forward-client" — matches the
# script and README's routing table: everything not matched by a rule below,
# including /oauth/callback, goes to the client SPA), or a fixed 404
# ("fixed-404" — what the live dev listener actually does; its client
# routing lives in hand-made path-pattern rules that Terraform does not yet
# manage). See var.https_default_action.
resource "aws_lb_listener" "https" {
  load_balancer_arn = aws_lb.this.arn
  port              = 443
  protocol          = "HTTPS"
  certificate_arn   = var.acm_certificate_arn

  dynamic "default_action" {
    for_each = var.https_default_action == "forward-client" ? [1] : []
    content {
      type             = "forward"
      target_group_arn = aws_lb_target_group.client.arn
    }
  }

  dynamic "default_action" {
    for_each = var.https_default_action == "fixed-404" ? [1] : []
    content {
      type = "fixed-response"

      fixed_response {
        content_type = "text/plain"
        message_body = var.fixed_404_message_body
        status_code  = "404"
      }
    }
  }

  tags = var.tags
}

# /api -> server. create-alb.example.sh's create_rule 10 '/api*' by
# default; the live dev ALB has this rule at priority 100 matching
# '/api/*' instead — see var.api_rule_priority / var.api_path_patterns.
resource "aws_lb_listener_rule" "api" {
  listener_arn = aws_lb_listener.https.arn
  priority     = var.api_rule_priority

  # Written in the expanded forward-block form (equivalent to
  # target_group_arn for a single TG) because that's what ELBv2 stores:
  # importing the live rule with the shorthand leaves a perpetual
  # stickiness/weight diff in plan.
  action {
    type = "forward"

    forward {
      target_group {
        arn    = aws_lb_target_group.server.arn
        weight = 1
      }

      stickiness {
        enabled  = false
        duration = 3600
      }
    }
  }

  condition {
    path_pattern {
      values = var.api_path_patterns
    }
  }
}

# Extra HTTPS path rules (e.g. live-dev client SPA routing at priorities
# 10–14). Keyed by priority so `terraform import
# 'module.alb.aws_lb_listener_rule.extra["10"]' <arn>` works. Default
# action / "default" rule is still the listener's default_action, not a
# listener_rule resource.
resource "aws_lb_listener_rule" "extra" {
  for_each = { for r in var.additional_https_rules : tostring(r.priority) => r }

  listener_arn = aws_lb_listener.https.arn
  priority     = each.value.priority

  action {
    type = "forward"

    forward {
      target_group {
        arn = (
          each.value.target == "server"
          ? aws_lb_target_group.server.arn
          : aws_lb_target_group.client.arn
        )
        weight = 1
      }

      stickiness {
        enabled  = false
        duration = 3600
      }
    }
  }

  condition {
    path_pattern {
      values = each.value.path_patterns
    }
  }
}

# /health (exact) -> server. Matches create-alb.example.sh's create_rule 20
# '/health'. The live dev listener has no such rule (its target groups
# still health-check /health directly against ports 8080/8081, which needs
# no listener rule) — var.create_health_rule = false skips it there.
resource "aws_lb_listener_rule" "health" {
  count = var.create_health_rule ? 1 : 0

  listener_arn = aws_lb_listener.https.arn
  priority     = var.health_rule_priority

  # Same expanded forward-block form as the api rule above, for the same
  # import-diff reason.
  action {
    type = "forward"

    forward {
      target_group {
        arn    = aws_lb_target_group.server.arn
        weight = 1
      }

      stickiness {
        enabled  = false
        duration = 3600
      }
    }
  }

  condition {
    path_pattern {
      values = ["/health"]
    }
  }
}

# HTTP -> HTTPS redirect — matches create-alb.example.sh's final listener.
resource "aws_lb_listener" "http_redirect" {
  load_balancer_arn = aws_lb.this.arn
  port              = 80
  protocol          = "HTTP"

  default_action {
    type = "redirect"

    redirect {
      port        = "443"
      protocol    = "HTTPS"
      status_code = "HTTP_301"
    }
  }

  tags = var.tags
}
