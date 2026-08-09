#!/usr/bin/env bash
# Example AWS CLI commands for ALB path routing (single EC2).
# Set variables, then run each section in order (AWS CloudShell or local CLI).
#
# EC2 docker-compose exposes:
#   client → host port 8080
#   server → host port 8081
#
# ALB listener rules (lower priority number = evaluated first):
#   10 /api*    → server target group (8081)
#   20 /health  → server target group (exact)
#   default     → client target group (8080) — includes /oauth/callback SPA route

set -euo pipefail

: "${AWS_REGION:=us-east-1}"
: "${VPC_ID:=vpc-xxxxxxxx}"
: "${SUBNET_IDS:=subnet-aaa,subnet-bbb}"   # comma-separated public subnets (ALB)
: "${EC2_INSTANCE_ID:=i-xxxxxxxx}"
: "${ACM_CERT_ARN:=arn:aws:acm:${AWS_REGION}:123456789012:certificate/xxxxxxxx}"
: "${ALB_SG_ID:=sg-xxxxxxxx}"              # allow 80/443 from 0.0.0.0/0
: "${ALB_NAME:=e2eetext}"

IFS=',' read -r -a SUBNET_ARRAY <<< "$SUBNET_IDS"

echo "Creating server target group (port 8081, health /health)..."
SERVER_TG_ARN="$(aws elbv2 create-target-group \
  --name "${ALB_NAME}-server" \
  --protocol HTTP \
  --port 8081 \
  --vpc-id "$VPC_ID" \
  --target-type instance \
  --health-check-path /health \
  --health-check-interval-seconds 30 \
  --query 'TargetGroups[0].TargetGroupArn' \
  --output text)"

echo "Creating client target group (port 8080)..."
CLIENT_TG_ARN="$(aws elbv2 create-target-group \
  --name "${ALB_NAME}-client" \
  --protocol HTTP \
  --port 8080 \
  --vpc-id "$VPC_ID" \
  --target-type instance \
  --health-check-path /health \
  --health-check-interval-seconds 30 \
  --query 'TargetGroups[0].TargetGroupArn' \
  --output text)"

aws elbv2 register-targets \
  --target-group-arn "$SERVER_TG_ARN" \
  --targets "Id=${EC2_INSTANCE_ID},Port=8081"

aws elbv2 register-targets \
  --target-group-arn "$CLIENT_TG_ARN" \
  --targets "Id=${EC2_INSTANCE_ID},Port=8080"

echo "Creating internet-facing ALB..."
ALB_ARN="$(aws elbv2 create-load-balancer \
  --name "$ALB_NAME" \
  --type application \
  --scheme internet-facing \
  --security-groups "$ALB_SG_ID" \
  --subnets "${SUBNET_ARRAY[@]}" \
  --query 'LoadBalancers[0].LoadBalancerArn' \
  --output text)"

aws elbv2 modify-load-balancer-attributes \
  --load-balancer-arn "$ALB_ARN" \
  --attributes Key=idle_timeout.timeout_seconds,Value=3600

echo "Creating HTTPS listener (default → client)..."
LISTENER_ARN="$(aws elbv2 create-listener \
  --load-balancer-arn "$ALB_ARN" \
  --protocol HTTPS \
  --port 443 \
  --certificates CertificateArn="$ACM_CERT_ARN" \
  --default-actions "Type=forward,TargetGroupArn=${CLIENT_TG_ARN}" \
  --query 'Listeners[0].ListenerArn' \
  --output text)"

create_rule() {
  local priority="$1"
  local pattern="$2"
  aws elbv2 create-rule \
    --listener-arn "$LISTENER_ARN" \
    --priority "$priority" \
    --conditions "Field=path-pattern,Values=${pattern}" \
    --actions "Type=forward,TargetGroupArn=${SERVER_TG_ARN}"
}

create_rule 10 '/api*'
create_rule 20 '/health'

echo "Creating HTTP → HTTPS redirect listener..."
aws elbv2 create-listener \
  --load-balancer-arn "$ALB_ARN" \
  --protocol HTTP \
  --port 80 \
  --default-actions 'Type=redirect,RedirectConfig={Protocol=HTTPS,Port=443,StatusCode=HTTP_301}'

echo "Done."
echo "ALB DNS: $(aws elbv2 describe-load-balancers --load-balancer-arns "$ALB_ARN" --query 'LoadBalancers[0].DNSName' --output text)"
echo "Point your domain (Route 53 alias or CNAME) at that DNS name."
echo "EC2 security group: allow TCP 8080 and 8081 inbound from ${ALB_SG_ID} only."
