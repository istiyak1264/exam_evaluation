###############################################################################
# Network
###############################################################################

resource "aws_vpc" "main" {
  cidr_block           = var.vpc_cidr
  enable_dns_hostnames = true
  enable_dns_support   = true
  tags                 = merge(var.tags, { Name = "${var.project}-vpc" })
}

resource "aws_internet_gateway" "main" {
  vpc_id = aws_vpc.main.id
  tags   = merge(var.tags, { Name = "${var.project}-igw" })
}

resource "aws_subnet" "public" {
  count                   = length(var.azs)
  vpc_id                  = aws_vpc.main.id
  cidr_block              = cidrsubnet(var.vpc_cidr, 8, count.index)
  availability_zone       = var.azs[count.index]
  map_public_ip_on_launch = true
  tags                    = merge(var.tags, { Name = "${var.project}-public-${count.index}" })
}

resource "aws_subnet" "private" {
  count             = length(var.azs)
  vpc_id            = aws_vpc.main.id
  cidr_block        = cidrsubnet(var.vpc_cidr, 8, count.index + 10)
  availability_zone = var.azs[count.index]
  tags              = merge(var.tags, { Name = "${var.project}-private-${count.index}" })
}

resource "aws_eip" "nat" {
  domain = "vpc"
  tags   = merge(var.tags, { Name = "${var.project}-nat-eip" })
}

resource "aws_nat_gateway" "main" {
  allocation_id = aws_eip.nat.id
  subnet_id     = aws_subnet.public[0].id
  depends_on    = [aws_internet_gateway.main]
  tags          = merge(var.tags, { Name = "${var.project}-nat" })
}

resource "aws_route_table" "public" {
  vpc_id = aws_vpc.main.id
  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.main.id
  }
  tags = merge(var.tags, { Name = "${var.project}-public-rt" })
}

resource "aws_route_table_association" "public" {
  count          = length(var.azs)
  subnet_id      = aws_subnet.public[count.index].id
  route_table_id = aws_route_table.public.id
}

resource "aws_route_table" "private" {
  vpc_id = aws_vpc.main.id
  route {
    cidr_block     = "0.0.0.0/0"
    nat_gateway_id = aws_nat_gateway.main.id
  }
  tags = merge(var.tags, { Name = "${var.project}-private-rt" })
}

resource "aws_route_table_association" "private" {
  count          = length(var.azs)
  subnet_id      = aws_subnet.private[count.index].id
  route_table_id = aws_route_table.private.id
}

###############################################################################
# Security groups — web -> backend -> db, plus ALB from the internet
###############################################################################

resource "aws_security_group" "alb" {
  name_prefix = "${var.project}-alb-"
  description = "Public entry point"
  vpc_id      = aws_vpc.main.id

  dynamic "ingress" {
    for_each = var.admin_cidr == "" ? [80, 443] : []
    content {
      from_port   = ingress.value
      to_port     = ingress.value
      protocol    = "tcp"
      cidr_blocks = ["0.0.0.0/0"]
    }
  }
  dynamic "ingress" {
    for_each = var.admin_cidr == "" ? [] : [80, 443]
    content {
      from_port   = ingress.value
      to_port     = ingress.value
      protocol    = "tcp"
      cidr_blocks = [var.admin_cidr]
    }
  }
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
  tags = merge(var.tags, { Name = "${var.project}-alb-sg" })

  lifecycle { create_before_destroy = true }
}

resource "aws_security_group" "web" {
  name_prefix = "${var.project}-web-"
  description = "nginx containers"
  vpc_id      = aws_vpc.main.id

  ingress {
    description     = "From the load balancer only"
    from_port       = 8080
    to_port         = 8080
    protocol        = "tcp"
    security_groups = [aws_security_group.alb.id]
  }
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
  tags = merge(var.tags, { Name = "${var.project}-web-sg" })

  lifecycle { create_before_destroy = true }
}

resource "aws_security_group" "backend" {
  name_prefix = "${var.project}-api-"
  description = "Go API containers"
  vpc_id      = aws_vpc.main.id

  ingress {
    description     = "From nginx"
    from_port       = 8080
    to_port         = 8080
    protocol        = "tcp"
    security_groups = [aws_security_group.web.id]
  }
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
  tags = merge(var.tags, { Name = "${var.project}-api-sg" })

  lifecycle { create_before_destroy = true }
}

resource "aws_security_group" "db" {
  name_prefix = "${var.project}-db-"
  description = "MySQL"
  vpc_id      = aws_vpc.main.id

  ingress {
    description     = "From the API containers"
    from_port       = 3306
    to_port         = 3306
    protocol        = "tcp"
    security_groups = [aws_security_group.backend.id]
  }
  tags = merge(var.tags, { Name = "${var.project}-db-sg" })

  lifecycle { create_before_destroy = true }
}

###############################################################################
# Storage: script PDFs
###############################################################################

resource "aws_kms_key" "scripts" {
  description             = "${var.project} script PDFs at rest"
  deletion_window_in_days = 30
  enable_key_rotation     = true
  tags                    = var.tags
}

resource "aws_s3_bucket" "scripts" {
  bucket        = "${var.project}-scripts-${data.aws_caller_identity.current.account_id}"
  force_destroy = false
  tags          = var.tags
}

resource "aws_s3_bucket_versioning" "scripts" {
  bucket = aws_s3_bucket.scripts.id
  versioning_configuration { status = "Enabled" }
}

resource "aws_s3_bucket_public_access_block" "scripts" {
  bucket                  = aws_s3_bucket.scripts.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_server_side_encryption_configuration" "scripts" {
  bucket = aws_s3_bucket.scripts.id
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm     = "aws:kms"
      kms_master_key_id = aws_kms_key.scripts.arn
    }
    bucket_key_enabled = true
  }
}

resource "aws_s3_bucket_lifecycle_configuration" "scripts" {
  bucket = aws_s3_bucket.scripts.id
  rule {
    id     = "archive-old-scripts"
    status = "Enabled"
    filter {}
    transition {
      days          = 365
      storage_class = "GLACIER_IR"
    }
    noncurrent_version_expiration { noncurrent_days = 90 }
  }
}

###############################################################################
# Database
###############################################################################

resource "aws_db_subnet_group" "main" {
  name       = "${var.project}-db"
  subnet_ids = aws_subnet.private[*].id
  tags       = var.tags
}

resource "random_password" "db" {
  length           = 32
  special          = true
  override_special = "!#%*-_=+"
}

resource "aws_db_parameter_group" "main" {
  name   = "${var.project}-mysql84"
  family = "mysql8.4"

  parameter {
    name  = "character_set_server"
    value = "utf8mb4"
  }
  parameter {
    name  = "collation_server"
    value = "utf8mb4_0900_ai_ci"
  }
  parameter {
    name  = "require_secure_transport"
    value = "ON"
  }
  tags = var.tags
}

resource "aws_db_instance" "main" {
  identifier                   = "${var.project}-db"
  engine                       = "mysql"
  engine_version               = "8.4"
  instance_class               = var.db_instance_class
  allocated_storage            = 50
  max_allocated_storage        = 500
  storage_type                 = "gp3"
  storage_encrypted            = true
  db_name                      = var.db_name
  username                     = var.db_username
  password                     = random_password.db.result
  parameter_group_name         = aws_db_parameter_group.main.name
  db_subnet_group_name         = aws_db_subnet_group.main.name
  vpc_security_group_ids       = [aws_security_group.db.id]
  multi_az                     = var.db_multi_az
  backup_retention_period      = 14
  backup_window                = "01:00-02:00"
  maintenance_window           = "sun:03:00-sun:04:00"
  delete_automated_backups     = false
  deletion_protection          = var.db_deletion_protection
  performance_insights_enabled = true
  auto_minor_version_upgrade   = true
  skip_final_snapshot          = false
  final_snapshot_identifier    = "${var.project}-final"

  tags = var.tags
}

###############################################################################
# Secrets
###############################################################################

resource "random_password" "jwt" {
  length  = 48
  special = false
}

resource "aws_secretsmanager_secret" "app" {
  name                    = "${var.project}/app"
  recovery_window_in_days = 7
  tags                    = var.tags
}

resource "aws_secretsmanager_secret_version" "app" {
  secret_id = aws_secretsmanager_secret.app.id
  secret_string = jsonencode({
    JWT_SECRET              = random_password.jwt.result
    BOOTSTRAP_ADMIN_EMAIL   = "registrar@${var.project}.example"
    BOOTSTRAP_ADMIN_PASSWORD = "ChangeMe#${random_password.jwt.result}"
  })
}

# Full DSN, built after the instance so the endpoint is known.
resource "aws_secretsmanager_secret" "db_url" {
  name                    = "${var.project}/db-url"
  recovery_window_in_days = 7
  tags                    = var.tags
}

resource "aws_secretsmanager_secret_version" "db_url" {
  secret_id = aws_secretsmanager_secret.db_url.id
  secret_string = format(
    "%s:%s@tcp(%s:3306)/%s?parseTime=true&multiStatements=true&tls=skip-verify&timeout=10s",
    var.db_username, random_password.db.result, aws_db_instance.main.address, var.db_name,
  )
}

###############################################################################
# Container registry + logs
###############################################################################

resource "aws_ecr_repository" "backend" {
  name                 = "${var.project}/backend"
  image_tag_mutability = "MUTABLE"
  image_scanning_configuration { scan_on_push = true }
  encryption_configuration { encryption_type = "AES256" }
  tags = var.tags
}

resource "aws_ecr_repository" "web" {
  name                 = "${var.project}/web"
  image_tag_mutability = "MUTABLE"
  image_scanning_configuration { scan_on_push = true }
  encryption_configuration { encryption_type = "AES256" }
  tags = var.tags
}

resource "aws_cloudwatch_log_group" "backend" {
  name              = "/ecs/${var.project}/backend"
  retention_in_days = var.log_retention_days
  tags              = var.tags
}

resource "aws_cloudwatch_log_group" "web" {
  name              = "/ecs/${var.project}/web"
  retention_in_days = var.log_retention_days
  tags              = var.tags
}

###############################################################################
# Load balancer
###############################################################################

resource "aws_lb" "main" {
  name               = "${var.project}-alb"
  internal           = false
  load_balancer_type = "application"
  security_groups    = [aws_security_group.alb.id]
  subnets            = aws_subnet.public[*].id
  idle_timeout       = 120

  access_logs {
    bucket  = aws_s3_bucket.logs.id
    prefix  = "alb"
    enabled = true
  }

  depends_on = [aws_s3_bucket_policy.logs]
  tags       = var.tags
}

resource "aws_lb_target_group" "web" {
  name        = "${var.project}-web"
  port        = 8080
  protocol    = "HTTP"
  vpc_id      = aws_vpc.main.id
  target_type = "ip"

  health_check {
    path                = "/"
    matcher             = "200"
    interval            = 30
    timeout             = 5
    healthy_threshold   = 2
    unhealthy_threshold = 3
  }
  tags = var.tags
}

resource "aws_lb_listener" "http" {
  load_balancer_arn = aws_lb.main.arn
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
}

resource "aws_lb_listener" "https" {
  count             = var.acm_certificate_arn == "" ? 0 : 1
  load_balancer_arn = aws_lb.main.arn
  port              = 443
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-TLS13-1-2-2021-06"
  certificate_arn   = var.acm_certificate_arn

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.web.arn
  }
}

# When no certificate is configured, HTTP forwards directly.
resource "aws_lb_listener_rule" "http_forward" {
  count        = var.acm_certificate_arn == "" ? 1 : 0
  listener_arn = aws_lb_listener.http.arn
  priority     = 1

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.web.arn
  }
  condition {
    path_pattern { values = ["/*"] }
  }
}

###############################################################################
# ECS
###############################################################################

resource "aws_ecs_cluster" "main" {
  name = var.project

  setting {
    name  = "containerInsights"
    value = "enabled"
  }
  tags = var.tags
}

resource "aws_iam_role" "exec" {
  name = "${var.project}-exec"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "ecs-tasks.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
  tags = var.tags
}

resource "aws_iam_role_policy_attachment" "exec" {
  role       = aws_iam_role.exec.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

resource "aws_iam_role" "task" {
  name = "${var.project}-task"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "ecs-tasks.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
  tags = var.tags
}

resource "aws_iam_role_policy" "task_s3" {
  name = "scripts-bucket"
  role = aws_iam_role.task.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["s3:PutObject", "s3:GetObject", "s3:DeleteObject"]
        Resource = "${aws_s3_bucket.scripts.arn}/*"
      },
      {
        Effect   = "Allow"
        Action   = ["kms:Decrypt", "kms:GenerateDataKey"]
        Resource = aws_kms_key.scripts.arn
      }
    ]
  })
}

resource "aws_iam_policy" "read_app_secret" {
  name = "${var.project}-read-app-secret"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action = ["secretsmanager:GetSecretValue"]
      Resource = [
        aws_secretsmanager_secret.app.arn,
        aws_secretsmanager_secret.db_url.arn,
      ]
    }]
  })
}

resource "aws_iam_role_policy_attachment" "exec_secret" {
  role       = aws_iam_role.exec.name
  policy_arn = aws_iam_policy.read_app_secret.arn
}

resource "aws_ecs_task_definition" "backend" {
  family                   = "${var.project}-backend"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = var.backend_cpu
  memory                   = var.backend_memory
  execution_role_arn       = aws_iam_role.exec.arn
  task_role_arn            = aws_iam_role.task.arn

  container_definitions = jsonencode([{
    name      = "backend"
    image     = "${aws_ecr_repository.backend.repository_url}:${var.image_tag}"
    essential = true
    portMappings = [{ containerPort = 8080, protocol = "tcp" }]
    environment = [
      { name = "APP_ENV", value = "production" },
      { name = "ADDR", value = ":8080" },
      { name = "STORAGE_DRIVER", value = "s3" },
      { name = "S3_BUCKET", value = aws_s3_bucket.scripts.id },
      { name = "S3_REGION", value = var.region },
      { name = "COOKIE_SECURE", value = "true" },
      { name = "ALLOWED_ORIGINS", value = "https://${var.project}.example" },
      { name = "MAX_UPLOAD_MB", value = "25" },
    ]
    secrets = [
      { name = "JWT_SECRET", valueFrom = "${aws_secretsmanager_secret.app.arn}:JWT_SECRET::" },
      { name = "BOOTSTRAP_ADMIN_EMAIL", valueFrom = "${aws_secretsmanager_secret.app.arn}:BOOTSTRAP_ADMIN_EMAIL::" },
      { name = "BOOTSTRAP_ADMIN_PASSWORD", valueFrom = "${aws_secretsmanager_secret.app.arn}:BOOTSTRAP_ADMIN_PASSWORD::" },
      { name = "DATABASE_URL", valueFrom = aws_secretsmanager_secret.db_url.arn },
    ]
    logConfiguration = {
      logDriver = "awslogs"
      options = {
        awslogs-group         = aws_cloudwatch_log_group.backend.name
        awslogs-region        = var.region
        awslogs-stream-prefix = "api"
      }
    }
    healthCheck = {
      command     = ["CMD-SHELL", "wget -qO- http://127.0.0.1:8080/healthz || exit 1"]
      interval    = 30
      timeout     = 5
      retries     = 3
      startPeriod = 40
    }
  }])

  tags = var.tags
}

resource "aws_ecs_task_definition" "web" {
  family                   = "${var.project}-web"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = 256
  memory                   = 512
  execution_role_arn       = aws_iam_role.exec.arn

  container_definitions = jsonencode([{
    name      = "web"
    image     = "${aws_ecr_repository.web.repository_url}:${var.image_tag}"
    essential = true
    portMappings = [{ containerPort = 8080, protocol = "tcp" }]
    logConfiguration = {
      logDriver = "awslogs"
      options = {
        awslogs-group         = aws_cloudwatch_log_group.web.name
        awslogs-region        = var.region
        awslogs-stream-prefix = "web"
      }
    }
  }])

  tags = var.tags
}

resource "aws_ecs_service" "backend" {
  name            = "backend"
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.backend.arn
  desired_count   = var.backend_desired_count
  launch_type     = "FARGATE"

  network_configuration {
    subnets          = aws_subnet.private[*].id
    security_groups  = [aws_security_group.backend.id]
    assign_public_ip = false
  }

  deployment_circuit_breaker {
    enable   = true
    rollback = true
  }

  deployment_minimum_healthy_percent = 100
  deployment_maximum_percent         = 200

  tags = var.tags
}

resource "aws_ecs_service" "web" {
  name            = "web"
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.web.arn
  desired_count   = var.web_desired_count
  launch_type     = "FARGATE"

  network_configuration {
    subnets          = aws_subnet.private[*].id
    security_groups  = [aws_security_group.web.id]
    assign_public_ip = false
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.web.arn
    container_name   = "web"
    container_port   = 8080
  }

  deployment_circuit_breaker {
    enable   = true
    rollback = true
  }

  depends_on = [aws_lb_listener.http]
  tags       = var.tags
}

data "aws_caller_identity" "current" {}

###############################################################################
# ALB access logging
###############################################################################

resource "aws_s3_bucket" "logs" {
  bucket        = "${var.project}-alb-logs-${data.aws_caller_identity.current.account_id}"
  force_destroy = true
  tags          = var.tags
}

resource "aws_s3_bucket_public_access_block" "logs" {
  bucket                  = aws_s3_bucket.logs.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_policy" "logs" {
  bucket = aws_s3_bucket.logs.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { AWS = "arn:aws:iam::127311923021:root" } # eu-west-1 ELB account
      Action    = "s3:PutObject"
      Resource  = "${aws_s3_bucket.logs.arn}/alb/*"
    }]
  })
}
