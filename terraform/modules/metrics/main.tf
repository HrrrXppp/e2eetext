# Scheduled Lambda that publishes dataset-wide Messages / Chats / Users counts
# to CloudWatch (#37). There is deliberately no function URL, API Gateway or
# any other invocation path: only the EventBridge rule below can trigger it,
# so nothing is reachable from outside AWS.

locals {
  function_name = "${var.name}-metrics"

  # The Lambda is always a source. EC2 security groups are required as well:
  # private DNS on the monitoring endpoint steers the server's PutMetricData
  # at this endpoint, and a Lambda-only ingress rule drops that traffic.
  endpoint_ingress_security_group_ids = distinct(concat(
    [aws_security_group.lambda.id],
    var.endpoint_ingress_security_group_ids,
  ))
}

data "aws_vpc" "this" {
  id = var.vpc_id
}

data "aws_region" "current" {}

# --- Networking ---

resource "aws_security_group" "lambda" {
  name        = "${local.function_name}-lambda"
  description = "Metrics Lambda: egress to RDS and VPC endpoints only"
  vpc_id      = var.vpc_id

  egress {
    description = "PostgreSQL within the VPC"
    from_port   = 5432
    to_port     = 5432
    protocol    = "tcp"
    cidr_blocks = [data.aws_vpc.this.cidr_block]
  }

  egress {
    description = "HTTPS to VPC endpoints"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = [data.aws_vpc.this.cidr_block]
  }

  tags = var.tags
}

resource "aws_security_group_rule" "rds_from_lambda" {
  count = var.rds_security_group_id != "" ? 1 : 0

  type                     = "ingress"
  description              = "PostgreSQL from the metrics Lambda"
  from_port                = 5432
  to_port                  = 5432
  protocol                 = "tcp"
  security_group_id        = var.rds_security_group_id
  source_security_group_id = aws_security_group.lambda.id
}

resource "aws_security_group" "endpoints" {
  count = var.create_vpc_endpoints ? 1 : 0

  name        = "${local.function_name}-endpoints"
  description = "Interface VPC endpoints used by the metrics Lambda"
  vpc_id      = var.vpc_id

  # One permission with every source group. A for_each keyed on the
  # Lambda security group id is unknown until apply and cannot be planned.
  ingress {
    description     = "HTTPS from the metrics Lambda and instances that publish CloudWatch metrics"
    from_port       = 443
    to_port         = 443
    protocol        = "tcp"
    security_groups = local.endpoint_ingress_security_group_ids
  }

  tags = var.tags
}

resource "aws_vpc_endpoint" "interface" {
  for_each = var.create_vpc_endpoints ? toset(["monitoring", "secretsmanager"]) : toset([])

  vpc_id              = var.vpc_id
  service_name        = "com.amazonaws.${data.aws_region.current.region}.${each.key}"
  vpc_endpoint_type   = "Interface"
  subnet_ids          = var.subnet_ids
  security_group_ids  = [aws_security_group.endpoints[0].id]
  private_dns_enabled = true

  tags = merge(var.tags, { Name = "${local.function_name}-${each.key}" })
}

# --- IAM ---

data "aws_iam_policy_document" "assume_role" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "lambda" {
  name               = local.function_name
  assume_role_policy = data.aws_iam_policy_document.assume_role.json
  tags               = var.tags
}

# CloudWatch Logs + the ENI permissions a VPC-attached Lambda needs.
resource "aws_iam_role_policy_attachment" "vpc_access" {
  role       = aws_iam_role.lambda.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaVPCAccessExecutionRole"
}

data "aws_iam_policy_document" "lambda" {
  statement {
    sid       = "PutMetricsToNamespace"
    effect    = "Allow"
    actions   = ["cloudwatch:PutMetricData"]
    resources = ["*"]

    condition {
      test     = "StringEquals"
      variable = "cloudwatch:namespace"
      values   = [var.namespace]
    }
  }

  statement {
    sid       = "ReadDatabaseSecret"
    effect    = "Allow"
    actions   = ["secretsmanager:GetSecretValue"]
    resources = [var.database_url_secret_arn]
  }
}

resource "aws_iam_role_policy" "lambda" {
  name   = "metrics"
  role   = aws_iam_role.lambda.id
  policy = data.aws_iam_policy_document.lambda.json
}

# --- Lambda ---

resource "aws_cloudwatch_log_group" "lambda" {
  name              = "/aws/lambda/${local.function_name}"
  retention_in_days = var.log_retention_days
  tags              = var.tags
}

resource "aws_lambda_function" "this" {
  function_name = local.function_name
  role          = aws_iam_role.lambda.arn

  runtime       = "provided.al2023"
  architectures = ["arm64"]
  handler       = "bootstrap"
  timeout       = 30
  memory_size   = 128

  filename         = var.lambda_zip_path
  source_code_hash = filebase64sha256(var.lambda_zip_path)

  environment {
    variables = {
      METRICS_NAMESPACE      = var.namespace
      METRICS_ENVIRONMENT    = var.environment
      DATABASE_URL_SECRET_ID = var.database_url_secret_arn
    }
  }

  vpc_config {
    subnet_ids         = var.subnet_ids
    security_group_ids = [aws_security_group.lambda.id]
  }

  tags = var.tags

  depends_on = [
    aws_cloudwatch_log_group.lambda,
    aws_iam_role_policy_attachment.vpc_access,
  ]
}

# --- Schedule (the only way to invoke the function) ---

resource "aws_cloudwatch_event_rule" "schedule" {
  name                = local.function_name
  description         = "Publish Messages/Chats/Users counts to CloudWatch"
  schedule_expression = var.schedule_expression
  tags                = var.tags
}

resource "aws_cloudwatch_event_target" "lambda" {
  rule = aws_cloudwatch_event_rule.schedule.name
  arn  = aws_lambda_function.this.arn
}

resource "aws_lambda_permission" "eventbridge" {
  statement_id  = "AllowEventBridge"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.this.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.schedule.arn
}
