# Publishing Terraform Package Provider to Terraform Registry

This guide covers publishing the `terraform-provider-pkg` to both the public Terraform Registry and private registries on Terraform Cloud.

## Table of Contents
- [Prerequisites](#prerequisites)
- [Public Registry Publishing](#public-registry-publishing)
- [Private Registry Publishing (Terraform Cloud)](#private-registry-publishing-terraform-cloud)
- [Verification](#verification)
- [Troubleshooting](#troubleshooting)

## Prerequisites

### Required Components
1. **GitHub Repository**: ✅ Already configured at `https://github.com/jamesainslie/terraform-package`
2. **GPG Signing Key**: ✅ Already configured
3. **GoReleaser Configuration**: ✅ Already configured in `.goreleaser.yml`
4. **GitHub Actions Workflow**: ✅ Already configured in `.github/workflows/release.yml`
5. **Terraform Cloud Account**: Required for private registry publishing

### Repository Structure Requirements
```
terraform-provider-pkg/
├── .github/
│   └── workflows/
│       └── release.yml          # Automated release workflow
├── .goreleaser.yml              # GoReleaser configuration
├── terraform-registry-manifest.json  # Registry metadata
├── docs/                        # Provider documentation
├── examples/                    # Usage examples
├── internal/                    # Provider implementation
├── go.mod                       # Go module definition
├── LICENSE                      # MIT License
└── README.md                    # Provider documentation
```

## Public Registry Publishing

### Step 1: Ensure Repository Naming Convention
Your repository must follow the naming pattern: `terraform-provider-{NAME}`
- ✅ Your repository: `terraform-provider-pkg` (where `pkg` is the provider name)

### Step 2: Create a GitHub Release

#### Automated Release (Recommended)
```bash
# Create a new tag
git tag -a v0.1.14 -m "Release v0.1.14 - Description of changes"

# Push the tag to trigger the release workflow
git push origin v0.1.14
```

The GitHub Actions workflow will automatically:
1. Run tests
2. Build binaries for all platforms
3. Create and sign release artifacts
4. Publish the GitHub release

#### Manual Release (Alternative)
```bash
# Ensure you have GoReleaser installed
brew install goreleaser

# Set up GPG key for signing
export GPG_FINGERPRINT="9C0C55E27B861DC39D65CCC199303C9288D70AEE"

# Run GoReleaser
goreleaser release --clean
```

### Step 3: Verify Release Assets
Each release must include:
- Binary archives for each platform:
  - `terraform-provider-pkg_{VERSION}_darwin_amd64.tar.gz`
  - `terraform-provider-pkg_{VERSION}_darwin_arm64.tar.gz`
  - `terraform-provider-pkg_{VERSION}_linux_amd64.tar.gz`
  - `terraform-provider-pkg_{VERSION}_linux_arm64.tar.gz`
  - `terraform-provider-pkg_{VERSION}_windows_amd64.tar.gz`
- Metadata files:
  - `terraform-provider-pkg_{VERSION}_manifest.json`
  - `terraform-provider-pkg_{VERSION}_SHA256SUMS`
  - `terraform-provider-pkg_{VERSION}_SHA256SUMS.sig`

### Step 4: Public Registry Discovery
The public Terraform Registry automatically discovers providers that meet the requirements:

1. **Wait for Discovery**: The registry scans GitHub periodically (1-24 hours)
2. **Check Registry**: Visit `https://registry.terraform.io/providers/jamesainslie/pkg`
3. **Manual Submission** (if needed):
   - Go to https://registry.terraform.io
   - Sign in with GitHub
   - Click "Publish Provider"
   - Select your repository
   - Follow the submission wizard

## Private Registry Publishing (Terraform Cloud)

### Step 1: Access Terraform Cloud Private Registry
Navigate to: https://app.terraform.io/app/jamesainslie/registry/private/providers

### Step 2: Connect GitHub Repository

1. **Click "Connect to VCS"**
   - Select GitHub as your VCS provider
   - Authorize Terraform Cloud to access your GitHub account

2. **Select Repository**
   - Choose `jamesainslie/terraform-package` from the list
   - Click "Connect and Continue"

### Step 3: Configure Provider Settings

1. **Provider Name**: `pkg`
2. **Provider Namespace**: `jamesainslie`
3. **GPG Key Configuration**:
   ```bash
   # Export your GPG public key
   gpg --armor --export 9C0C55E27B861DC39D65CCC199303C9288D70AEE > gpg-public-key.asc
   
   # Copy the contents and paste into Terraform Cloud
   cat gpg-public-key.asc
   ```

4. **Publishing Settings**:
   - **Automatic Publishing**: Enable for automatic releases on new tags
   - **Version Tags**: Use pattern `v*` (e.g., `v0.1.14`)

### Step 4: Configure GitHub Secrets (if not already done)

Add these secrets to your GitHub repository:
```bash
# Navigate to Settings > Secrets and variables > Actions
GPG_PRIVATE_KEY    # Your GPG private key (exported with --armor)
GPG_PASSPHRASE     # Your GPG key passphrase
```

To export your GPG private key:
```bash
gpg --armor --export-secret-keys 9C0C55E27B861DC39D65CCC199303C9288D70AEE
```

### Step 5: Publish to Private Registry

#### Option A: Automatic Publishing
Once configured, any new GitHub release will automatically publish to your private registry.

#### Option B: Manual Publishing
1. Go to your private registry page
2. Click "Publish new version"
3. Select the GitHub release/tag
4. Click "Publish version"

### Step 6: Configure Provider Authentication

For private registry usage, configure authentication:

```hcl
# ~/.terraformrc or terraform.rc
credentials "app.terraform.io" {
  token = "your-terraform-cloud-api-token"
}
```

Generate an API token:
1. Go to https://app.terraform.io/app/settings/tokens
2. Click "Create an API token"
3. Name it (e.g., "CLI Token")
4. Copy the token

## Verification

### Verify Public Registry Publishing

```bash
# Check if provider is available
terraform init -upgrade

# In your Terraform configuration
terraform {
  required_providers {
    pkg = {
      source  = "jamesainslie/pkg"
      version = "~> 0.1.13"
    }
  }
}
```

### Verify Private Registry Publishing

```hcl
# For private registry
terraform {
  required_providers {
    pkg = {
      source  = "app.terraform.io/jamesainslie/pkg"
      version = "~> 0.1.13"
    }
  }
}
```

### Check Provider Information

```bash
# List available versions
terraform providers registry.terraform.io/jamesainslie/pkg

# Show provider documentation
terraform providers schema -json | jq '.provider_schemas["registry.terraform.io/jamesainslie/pkg"]'
```

## Troubleshooting

### Common Issues and Solutions

#### 1. Provider Not Appearing in Public Registry
**Issue**: Provider doesn't appear after 24 hours
**Solution**:
- Verify repository name follows `terraform-provider-{name}` pattern
- Check all required release assets are present
- Ensure releases are not marked as draft or pre-release
- Manually submit via registry.terraform.io

#### 2. GPG Signature Verification Failed
**Issue**: Registry reports invalid GPG signature
**Solution**:
```bash
# Verify signature locally
gpg --verify terraform-provider-pkg_0.1.13_SHA256SUMS.sig terraform-provider-pkg_0.1.13_SHA256SUMS

# Re-sign if needed
gpg --detach-sign --armor terraform-provider-pkg_0.1.13_SHA256SUMS
```

#### 3. GoReleaser Build Failures
**Issue**: Release workflow fails during GoReleaser step
**Solution**:
- Check `.goreleaser.yml` configuration
- Ensure no duplicate files in archives
- Verify binary naming patterns
- Run locally to debug: `goreleaser release --snapshot --skip-publish --clean`

#### 4. Private Registry Connection Issues
**Issue**: Cannot connect GitHub repository to Terraform Cloud
**Solution**:
- Ensure GitHub App permissions are granted
- Repository must be public or you need appropriate Terraform Cloud plan
- Check organization settings allow GitHub integration

#### 5. Version Conflicts
**Issue**: Version already exists error
**Solution**:
```bash
# Delete the tag locally and remotely
git tag -d v0.1.14
git push origin :refs/tags/v0.1.14

# Create new tag with incremented version
git tag -a v0.1.15 -m "Release v0.1.15"
git push origin v0.1.15
```

## Best Practices

### Version Management
- Follow Semantic Versioning (MAJOR.MINOR.PATCH)
- Use `v` prefix for tags (e.g., `v0.1.14`)
- Document changes in CHANGELOG.md
- Never reuse version numbers

### Release Process
1. Update version in code if hardcoded
2. Update CHANGELOG.md
3. Commit changes
4. Create and push tag
5. Monitor GitHub Actions workflow
6. Verify release on GitHub
7. Check registry availability

### Documentation
- Keep README.md updated
- Maintain examples in `examples/` directory
- Document all resources and data sources
- Include migration guides for breaking changes

### Security
- Never commit GPG private keys
- Rotate GPG keys periodically
- Use GitHub Secrets for sensitive data
- Enable 2FA on GitHub and Terraform Cloud

## Monitoring and Maintenance

### Check Provider Usage
- Public Registry: View download statistics on registry.terraform.io
- Private Registry: Check usage in Terraform Cloud organization settings

### Update Schedule
- Security patches: Immediate
- Bug fixes: Within 1 week
- Features: Regular release cycle (e.g., monthly)

### Deprecation Policy
- Announce deprecations at least 2 versions in advance
- Maintain backward compatibility when possible
- Provide migration guides

## Additional Resources

- [Terraform Registry Documentation](https://www.terraform.io/docs/registry/providers/publishing.html)
- [GoReleaser Documentation](https://goreleaser.com/customization/)
- [Terraform Cloud Private Registry](https://www.terraform.io/docs/cloud/registry/index.html)
- [GitHub Actions for Terraform](https://github.com/hashicorp/setup-terraform)
- [Provider Development Best Practices](https://www.terraform.io/docs/extend/best-practices/index.html)

## Support

For issues or questions:
- Public Registry: Open issue at https://github.com/jamesainslie/terraform-package/issues
- Private Registry: Contact Terraform Cloud support
- Community: HashiCorp Discuss forums

---

Last Updated: September 21, 2025
Provider Version: v0.1.13
