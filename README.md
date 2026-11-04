# Secure Exam Evaluation System

Anonymised, double-blind marking of exam scripts for a university examinations office.
Scripts are registered under an opaque cover-sheet code, marked independently by two
examiners, and escalated to a third external examiner whenever their marks differ by
**13 or more**. Below that gap the two marks are averaged.

Go API · MySQL 8.4 · React (plain JavaScript, no TypeScript) · Docker · Terraform for AWS.

---

## The rule, in one place

`backend/internal/grading/grading.go` is the only place the policy lives, and it is
covered by table-driven tests:

| Situation | Outcome |
|---|---|
| gap `< 13` | `final = (m1 + m2) / 2`, script `RESOLVED` |
| gap `>= 13` | script moves to `THIRD_REVIEW`, third external examiner appointed |
| three marks | `final = median(m1, m2, m3)`, script `RESOLVED` |

The median survives an outlier in either direction, which is the point of the third
review. A single submission leaves the script pending; four marks are rejected.

## Roles

| Role | Can do |
|---|---|
| `ADMIN` | everything, plus users, stats, audit, board override |
| `HALL_MANAGER` | register scripts + PDFs, record the code↔student link, appoint examiners, search |
| `EVALUATOR` | see only assigned scripts, blinded; one mark per slot, no editing after submit |
| `STUDENT` | own results, own PDF, one challenge per script |

Blinding is enforced in the API, not the UI: the evaluator projection has no
`student_name`/`student_no`, and PDF access is checked against the assignment table.

## Data model

```
academic_sessions ─┬─ students ─┐
                   └─ courses ──┴─< scripts >── script_assignments
                                        │  └──< marks (FIRST/SECOND/THIRD)
                                        ├── results (final_marks, method, gap)
                                        └──< grade_challenges
users ──< refresh_tokens      audit_log (append-only)
```

`scripts` holds the only link to a student. `UNIQUE(course_id, student_id)` enforces one
script per student per course; `UNIQUE(script_id, slot)` enforces one mark per examiner
position.

## Run it

```bash
cp .env.example .env          # then edit JWT_SECRET
docker compose up -d --build  # db, api, web
docker compose --profile seed run --rm seed   # optional demo cohort
```

Web on <http://localhost:8090> (8080 left free for other local tooling; change `WEB_PORT`).

Seeded logins all use `ExamOffice#2026`:

| Account | Role |
|---|---|
| `admin@uni.test` | ADMIN |
| `registrar@uni.test` | HALL_MANAGER |
| `examiner.one@uni.test` | EVALUATOR |
| `adaeze@student.test` | STUDENT |

Change these before any real use.

## Tests

```bash
make test-unit   # grading policy, no database
make test-api    # full API suite against a throwaway MySQL container
make test        # both
```

The API suite covers authn/refresh rotation, role boundaries, the blind projection, the
average path, the exactly-13 escalation, median resolution, duplicate-mark conflicts,
PDF token scoping, the student portal, challenges, overrides and the audit trail.

## API

```
POST   /api/auth/login | //refresh | //logout      GET /api/auth/me
GET    /api/scripts            POST /api/scripts (multipart: file, course_id, student_id)
GET    /api/scripts/:id        POST /api/scripts/:id/assign | //assign-third | //override
POST   /api/scripts/:id/marks  POST /api/scripts/:id/pdf-token   GET /api/scripts/:id/pdf?t=
GET    /api/queue              GET  /api/results | //results/mine | //export/results.csv
GET    /api/challenges         POST /api/challenges             PATCH /api/challenges/:id
GET    /api/admin/stats | //audit
```

PDFs are served through short-lived (10 min) scoped tokens, because an `<iframe>` cannot
send an `Authorization` header.

## Security notes

- Passwords: bcrypt cost 12, minimum 10 characters. Login throttled per email+IP.
- Tokens: 15-minute access, 7-day rotating refresh stored as SHA-256 hashes.
- Every state change is written to `audit_log` with actor, role and metadata.
- Uploads validated by `%PDF-` magic, size capped, SHA-256 recorded, stored under an
  unguessable course/key path.
- `JWT_SECRET` must be ≥ 32 bytes in production or the server refuses to boot.
- Responses carry CSP, `nosniff`, `no-referrer`, `X-Frame-Options: SAMEORIGIN`.

## Deploy to AWS

`infra/terraform` provisions VPC (public ALB / private tasks and DB), ECS Fargate
services behind an ALB, RDS MySQL 8.4 with TLS required and 14-day backups, an S3 bucket
for PDFs (SSE-KMS, versioning, Glacier IR after a year), ECR, Secrets Manager and
CloudWatch logs.

```bash
cd infra/terraform
terraform init
terraform apply -var acm_certificate_arn=arn:aws:acm:...   # cert enables HTTPS
```

Then push images and roll the services:

```bash
aws ecr get-login-password | docker login --username AWS --password-stdin <acct>.dkr.ecr.<region>.amazonaws.com
docker build -t <ecr-backend>:$(git rev-parse --short HEAD) ./backend
docker build -t <ecr-web>:$(git rev-parse --short HEAD) ./frontend
docker push <ecr-backend>:$(git rev-parse --short HEAD)
docker push <ecr-web>:$(git rev-parse --short HEAD)
terraform apply -var image_tag=$(git rev-parse --short HEAD)
```

The bootstrap admin is created by the first task start from the secret; sign in and
change the password immediately.

## Layout

```
backend/cmd/server      API entrypoint (+ graceful shutdown, bootstrap admin)
backend/cmd/seed        demo cohort with generated PDFs
backend/internal/api    handlers, middleware, routing
backend/internal/store  SQL
backend/internal/grading policy + tests
frontend/src/pages      role-specific screens (.jsx)
infra/terraform         AWS infrastructure
```
