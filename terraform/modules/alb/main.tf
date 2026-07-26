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

resource "aws_lb" "this" {
  name               = var.name
  internal           = false
  load_balancer_type = "application"
  security_groups    = var.security_group_ids
  subnets            = var.subnet_ids

  # create-alb.example.sh sets this explicitly for /ws WebSocket support.
  idle_timeout = 3600

  tags = var.tags
}

# Port 8081, health check /health — matches create-alb.example.sh's
# "server target group" section exactly.
resource "aws_lb_target_group" "server" {
  name        = "${var.name}-server"
  port        = 8081
  protocol    = "HTTP"
  vpc_id      = var.vpc_id
  target_type = "instance"

  health_check {
    path     = "/health"
    interval = 30
  }

  tags = var.tags
}

# Port 8080, health check /health — matches create-alb.example.sh's
# "client target group" section exactly (yes, the client TG also uses
# /health as its health-check path per the script, even though it's the
# default/catch-all target rather than the one serving GET /health).
resource "aws_lb_target_group" "client" {
  name        = "${var.name}-client"
  port        = 8080
  protocol    = "HTTP"
  vpc_id      = var.vpc_id
  target_type = "instance"

  health_check {
    path     = "/health"
    interval = 30
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

# Default action forwards to the client TG — matches the script and
# README's routing table (everything not matched by a rule below, including
# /oauth/callback, goes to the client SPA).
resource "aws_lb_listener" "https" {
  load_balancer_arn = aws_lb.this.arn
  port              = 443
  protocol          = "HTTPS"
  certificate_arn   = var.acm_certificate_arn

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.client.arn
  }

  tags = var.tags
}

# Priority 10: /api* -> server. Matches create-alb.example.sh's create_rule
# 10 '/api*'.
resource "aws_lb_listener_rule" "api" {
  listener_arn = aws_lb_listener.https.arn
  priority     = 10

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.server.arn
  }

  condition {
    path_pattern {
      values = ["/api*"]
    }
  }
}

# Priority 20: /health (exact) -> server. Matches create-alb.example.sh's
# create_rule 20 '/health'.
resource "aws_lb_listener_rule" "health" {
  listener_arn = aws_lb_listener.https.arn
  priority     = 20

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.server.arn
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
