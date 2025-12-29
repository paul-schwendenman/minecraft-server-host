# GitHub Actions AWS OIDC Setup

GitHub Actions workflows use OpenID Connect (OIDC) to authenticate with AWS. This is more secure than storing long-lived AWS credentials as secrets.

## How It Works

1. GitHub Actions requests a short-lived OIDC token from GitHub's identity provider
2. AWS IAM validates the token against GitHub's OIDC provider
3. If valid, AWS issues temporary credentials to the workflow
4. Credentials expire after the workflow completes

## Setup Steps

### 1. Create the OIDC Identity Provider in AWS

Run this once per AWS account:

```bash
aws iam create-open-id-connect-provider \
  --url https://token.actions.githubusercontent.com \
  --client-id-list sts.amazonaws.com \
  --thumbprint-list 6938fd4d98bab03faadb97b34396831e3780aea1
```

Or via Terraform:

```hcl
resource "aws_iam_openid_connect_provider" "github" {
  url             = "https://token.actions.githubusercontent.com"
  client_id_list  = ["sts.amazonaws.com"]
  thumbprint_list = ["6938fd4d98bab03faadb97b34396831e3780aea1"]
}
```

### 2. Set Environment Variables

Set these variables once, then use `envsubst` to substitute them in the policy files:

```bash
export AWS_ACCOUNT_ID="YOUR_ACCOUNT_ID"
export GITHUB_REPO="YOUR_ORG/YOUR_REPO"

# Test environment
export TEST_LAMBDA_PREFIX="minecraft-test"        # Lambda functions: ${TEST_LAMBDA_PREFIX}-control, etc.
export TEST_S3_BUCKET="minecraft-test-webapp"     # S3 bucket for webapp
export TEST_CLOUDFRONT_DISTRIBUTION_ID="YOUR_TEST_DISTRIBUTION_ID"

# Prod environment (set when ready)
# export PROD_LAMBDA_PREFIX="minecraft-prod"
# export PROD_S3_BUCKET="minecraft-prod-webapp"
# export PROD_CLOUDFRONT_DISTRIBUTION_ID="YOUR_PROD_DISTRIBUTION_ID"
```

### 3. Create the IAM Role

Create a role that GitHub Actions can assume. The trust policy restricts access to your specific repository.

```bash
# Create the trust policy
cat << 'EOF' | envsubst | tee trust-policy.json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::${AWS_ACCOUNT_ID}:oidc-provider/token.actions.githubusercontent.com"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "token.actions.githubusercontent.com:aud": "sts.amazonaws.com"
        },
        "StringLike": {
          "token.actions.githubusercontent.com:sub": "repo:${GITHUB_REPO}:*"
        }
      }
    }
  ]
}
EOF

# Create the role
aws iam create-role \
  --role-name GitHubActionsRole \
  --assume-role-policy-document file://trust-policy.json
```

### 4. Attach Permissions for Packer

Packer needs permissions to create EC2 instances and AMIs:

```bash
cat << 'EOF' | tee packer-policy.json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "PackerEC2",
      "Effect": "Allow",
      "Action": [
        "ec2:AttachVolume",
        "ec2:AuthorizeSecurityGroupIngress",
        "ec2:CopyImage",
        "ec2:CreateImage",
        "ec2:CreateKeyPair",
        "ec2:CreateSecurityGroup",
        "ec2:CreateSnapshot",
        "ec2:CreateTags",
        "ec2:CreateVolume",
        "ec2:DeleteKeyPair",
        "ec2:DeleteSecurityGroup",
        "ec2:DeleteSnapshot",
        "ec2:DeleteVolume",
        "ec2:DeregisterImage",
        "ec2:DescribeImageAttribute",
        "ec2:DescribeImages",
        "ec2:DescribeInstances",
        "ec2:DescribeInstanceStatus",
        "ec2:DescribeRegions",
        "ec2:DescribeSecurityGroups",
        "ec2:DescribeSnapshots",
        "ec2:DescribeSubnets",
        "ec2:DescribeTags",
        "ec2:DescribeVolumes",
        "ec2:DetachVolume",
        "ec2:GetPasswordData",
        "ec2:ModifyImageAttribute",
        "ec2:ModifyInstanceAttribute",
        "ec2:ModifySnapshotAttribute",
        "ec2:RegisterImage",
        "ec2:RunInstances",
        "ec2:StopInstances",
        "ec2:TerminateInstances"
      ],
      "Resource": "*"
    }
  ]
}
EOF

aws iam put-role-policy \
  --role-name GitHubActionsRole \
  --policy-name PackerBuildPolicy \
  --policy-document file://packer-policy.json
```

### 5. Attach Permissions for Test Deployments

Lambda and webapp deployments to the test environment need these permissions:

```bash
cat << 'EOF' | envsubst | tee test-deploy-policy.json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "LambdaDeployTest",
      "Effect": "Allow",
      "Action": ["lambda:UpdateFunctionCode"],
      "Resource": "arn:aws:lambda:us-east-2:${AWS_ACCOUNT_ID}:function:${TEST_LAMBDA_PREFIX}-*"
    },
    {
      "Sid": "S3WebappDeployTest",
      "Effect": "Allow",
      "Action": [
        "s3:ListBucket",
        "s3:GetObject",
        "s3:PutObject",
        "s3:DeleteObject"
      ],
      "Resource": [
        "arn:aws:s3:::${TEST_S3_BUCKET}",
        "arn:aws:s3:::${TEST_S3_BUCKET}/*"
      ]
    },
    {
      "Sid": "CloudFrontInvalidateTest",
      "Effect": "Allow",
      "Action": ["cloudfront:CreateInvalidation"],
      "Resource": "arn:aws:cloudfront::${AWS_ACCOUNT_ID}:distribution/${TEST_CLOUDFRONT_DISTRIBUTION_ID}"
    }
  ]
}
EOF

aws iam put-role-policy \
  --role-name GitHubActionsRole \
  --policy-name TestDeployPolicy \
  --policy-document file://test-deploy-policy.json
```

### 6. Attach Permissions for Prod Deployments (when ready)

When you set up the production environment, set the prod distribution ID and add these permissions:

```bash
export PROD_LAMBDA_PREFIX="minecraft-prod"
export PROD_S3_BUCKET="minecraft-prod-webapp"
export PROD_CLOUDFRONT_DISTRIBUTION_ID="YOUR_PROD_DISTRIBUTION_ID"

cat << 'EOF' | envsubst | tee prod-deploy-policy.json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "LambdaDeployProd",
      "Effect": "Allow",
      "Action": ["lambda:UpdateFunctionCode"],
      "Resource": "arn:aws:lambda:us-east-2:${AWS_ACCOUNT_ID}:function:${PROD_LAMBDA_PREFIX}-*"
    },
    {
      "Sid": "S3WebappDeployProd",
      "Effect": "Allow",
      "Action": [
        "s3:ListBucket",
        "s3:GetObject",
        "s3:PutObject",
        "s3:DeleteObject"
      ],
      "Resource": [
        "arn:aws:s3:::${PROD_S3_BUCKET}",
        "arn:aws:s3:::${PROD_S3_BUCKET}/*"
      ]
    },
    {
      "Sid": "CloudFrontInvalidateProd",
      "Effect": "Allow",
      "Action": ["cloudfront:CreateInvalidation"],
      "Resource": "arn:aws:cloudfront::${AWS_ACCOUNT_ID}:distribution/${PROD_CLOUDFRONT_DISTRIBUTION_ID}"
    }
  ]
}
EOF

aws iam put-role-policy \
  --role-name GitHubActionsRole \
  --policy-name ProdDeployPolicy \
  --policy-document file://prod-deploy-policy.json
```

### 7. Add the Role ARN to GitHub Secrets

Replace:

- `ACCOUNT_ID` with your AWS account ID

1. Go to your repository on GitHub
2. Navigate to **Settings** > **Secrets and variables** > **Actions**
3. Click **New repository secret**
4. Name: `AWS_ROLE_ARN`
5. Value: `arn:aws:iam::ACCOUNT_ID:role/GitHubActionsRole`

## Terraform Module

The GitHub Actions IAM role is managed by Terraform in `infra/global/` using the `github_actions_role` module.

### Module Location

- **Module:** `infra/modules/github_actions_role/`
- **Usage:** `infra/global/main.tf`

### What the Module Creates

- OIDC identity provider for GitHub Actions
- IAM role with trust policy for your repository
- Dynamic policies for:
  - **S3:** Deploy to webapp buckets
  - **CloudFront:** Cache invalidation
  - **Lambda:** Function code updates
  - **EC2/AMI:** Packer builds (optional)

### Usage

The `infra/global/main.tf` reads outputs from test and prod environments via `terraform_remote_state` and automatically configures permissions for all S3 buckets and CloudFront distributions.

```bash
# After applying test and prod environments
cd infra/global
terraform init
terraform apply
```

### Updating Permissions

When you add new CloudFront distributions or S3 buckets in test/prod, re-run:

```bash
cd infra/global
terraform apply
```

This automatically updates the IAM policies.

### Importing Existing Resources

If the OIDC provider and role already exist:

```bash
cd infra/global
terraform import module.github_actions_role.aws_iam_openid_connect_provider.github \
  arn:aws:iam::ACCOUNT_ID:oidc-provider/token.actions.githubusercontent.com
terraform import module.github_actions_role.aws_iam_role.github_actions GitHubActionsRole
```

### Module Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `github_repo` | Repository in format `org/repo` | (required) |
| `s3_buckets` | List of S3 bucket names | `[]` |
| `cloudfront_distribution_arns` | List of CloudFront ARNs | `[]` |
| `lambda_prefixes` | List of Lambda prefixes | `[]` |
| `include_packer_permissions` | Include EC2/AMI permissions | `true` |

## Restricting Access Further

You can make the trust policy more restrictive:

```json
{
  "Condition": {
    "StringEquals": {
      "token.actions.githubusercontent.com:aud": "sts.amazonaws.com",
      "token.actions.githubusercontent.com:sub": "repo:YOUR_ORG/YOUR_REPO:ref:refs/heads/master"
    }
  }
}
```

This restricts access to only the `master` branch. Other options:

- `repo:org/repo:ref:refs/heads/*` - any branch
- `repo:org/repo:environment:production` - specific environment
- `repo:org/repo:pull_request` - pull requests only

## Troubleshooting

### "Not authorized to perform sts:AssumeRoleWithWebIdentity"

- Check the trust policy has the correct repository name
- Verify the OIDC provider thumbprint is correct
- Ensure `sts.amazonaws.com` is in the audience list

### "Could not find OIDC provider"

- Create the OIDC provider in AWS (step 1)
- Verify it's in the correct AWS region

### "AccessDeniedException" for lambda:UpdateFunctionCode

- Add the TestDeployPolicy (step 5) to the role
- Verify the Lambda function name matches `${TEST_LAMBDA_PREFIX}-*` pattern

### "AccessDenied" for S3 operations

- Add the TestDeployPolicy (step 5) to the role
- Verify the S3 bucket name matches `${TEST_S3_BUCKET}`

## References

- [GitHub Docs: Configuring OIDC in AWS](https://docs.github.com/en/actions/deployment/security-hardening-your-deployments/configuring-openid-connect-in-amazon-web-services)
- [AWS Docs: Creating OIDC Identity Providers](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_providers_create_oidc.html)
- [Packer AWS Builder Docs](https://developer.hashicorp.com/packer/integrations/hashicorp/amazon)
