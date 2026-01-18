# Terraform Expert

Specialized guidance for working in `infra/` - Terraform infrastructure code.

## Project Structure

```
infra/
├── test/           # Test environment (t3.small, 10.0.0.0/16)
├── prod/           # Production environment (t3.medium, 10.1.0.0/16)
├── global/         # Account-wide shared resources
├── minimal/        # Starter template
└── modules/        # Shared module library
    ├── networking, s3_buckets, ec2_role, mc_stack
    ├── api_lambda, web_ui, acm_certificate, dns_records
    └── github_actions_role
```

## Conventions

### Resource Naming
- Use `this` for primary resources: `aws_vpc.this`, `aws_instance.this`
- Construct names with prefix: `"${var.name}-vpc"`, `"${var.name}-igw"`
- Always include `Name` tag matching resource name

### Module Pattern
```hcl
# variables.tf - all inputs with description + type
variable "name" {
  description = "Resource name prefix"
  type        = string
}

# main.tf - resources using var.name prefix
resource "aws_vpc" "this" {
  cidr_block = var.cidr
  tags = { Name = "${var.name}-vpc" }
}

# outputs.tf - export key attributes
output "vpc_id" {
  value = aws_vpc.this.id
}
```

### Environment Differences
| Aspect | Test | Prod |
|--------|------|------|
| Instance | t3.small | t3.medium |
| VPC CIDR | 10.0.0.0/16 | 10.1.0.0/16 |
| Route53 | Data source (existing) | Resource (creates zone) |

### Multi-Region Pattern
```hcl
provider "aws" {
  region = "us-east-2"
}

provider "aws" {
  alias  = "us_east_1"
  region = "us-east-1"  # Required for CloudFront ACM certs
}
```

## Guidelines

1. **Prefer modules** - Don't duplicate; extend existing modules
2. **Cross-module refs** - Use module outputs, not `depends_on`
3. **No locals abuse** - Direct variable interpolation preferred
4. **AMI lookup** - Use data source for `minecraft-ubuntu-*` AMIs
5. **Sensitive vars** - Mark with `sensitive = true`

## Validation

```bash
cd infra/test  # or prod
terraform init
terraform validate
terraform plan
```
