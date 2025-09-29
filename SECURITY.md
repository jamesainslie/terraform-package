# Security Policy

## Supported Versions

We release patches for security vulnerabilities for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |
| < 0.1   | :x:                |

## Reporting a Vulnerability

We take security vulnerabilities seriously. Please follow responsible disclosure practices.

### How to Report

**DO NOT** create a public GitHub issue for security vulnerabilities.

Instead, please report security vulnerabilities by:

1. **Email**: Send details to the maintainers privately
2. **GitHub Security Advisories**: Use the "Security" tab in the repository
3. **Private discussion**: Contact maintainers directly

### What to Include

Please include the following information:

- **Description**: Clear description of the vulnerability
- **Impact**: Potential impact and affected components
- **Reproduction**: Steps to reproduce the vulnerability
- **Environment**: Operating system, Go version, Terraform version
- **Proposed fix**: If you have suggestions for fixing

### Response Timeline

- **Initial response**: Within 48 hours
- **Status update**: Within 7 days
- **Resolution timeline**: Depends on complexity, typically 14-30 days

## Security Measures

### Code Security

- **Input validation**: All user inputs are validated and sanitized
- **Command injection prevention**: Proper argument escaping
- **Privilege escalation**: Secure sudo/elevation handling
- **Timeout protection**: All operations have timeouts
- **Error handling**: No sensitive information in error messages

### Supply Chain Security

- **Dependency scanning**: Automated vulnerability scanning
- **Go module verification**: Checksums verified
- **Signed releases**: All releases are GPG signed
- **Reproducible builds**: Deterministic build process

### Runtime Security

- **Non-interactive mode**: Prevents interactive prompts
- **Privilege validation**: Checks privileges before operations
- **Resource limits**: Timeouts and resource constraints
- **Audit logging**: Security-relevant events are logged

## Security Best Practices for Users

### Provider Configuration

```hcl
provider "pkg" {
  # Enable non-interactive mode (recommended)
  assume_yes = true
  
  # Configure sudo carefully
  sudo_enabled = true  # Only if needed
  
  # Use specific paths to avoid PATH injection
  brew_path = "/opt/homebrew/bin/brew"
  
  # Set reasonable timeouts
  lock_timeout = "10m"
}
```

### Resource Security

```hcl
resource "pkg_package" "example" {
  # Use specific versions to avoid supply chain attacks
  version = "2.42.0"  # Exact version
  
  # Pin packages for stability
  pin = true
  
  # Set reasonable timeouts
  timeouts {
    create = "15m"
    delete = "10m"
  }
}
```

### Environment Security

- **Run with minimal privileges**: Don't run as root unless necessary
- **Validate package sources**: Use trusted repositories only
- **Monitor installations**: Use data sources to audit packages
- **Regular updates**: Keep provider and packages updated

## Known Security Considerations

### Package Manager Risks

- **Homebrew**: Generally safe, but casks can contain arbitrary software
- **APT**: Requires sudo, validate repository GPG keys
- **winget**: May require elevation, validate package sources
- **Chocolatey**: Community packages may have security risks

### Mitigation Strategies

1. **Repository validation**: Verify repository authenticity
2. **Version pinning**: Use specific versions in production
3. **Regular audits**: Use `pkg_installed_packages` data source
4. **Security scanning**: Monitor with `pkg_security_info` data source
5. **Privilege isolation**: Run with minimal required privileges

## Security Updates

### Update Process

1. **Security advisory** created for vulnerability
2. **Patch development** with security fix
3. **Testing** across all supported platforms
4. **Release** with security update
5. **Notification** to users via GitHub and registry

### Staying Updated

- **Watch the repository** for security advisories
- **Subscribe to releases** for update notifications
- **Monitor dependencies** with tools like Dependabot
- **Regular updates**: Update provider regularly

## Vulnerability Disclosure Timeline

1. **Day 0**: Vulnerability reported privately
2. **Day 1-2**: Initial response and acknowledgment
3. **Day 3-7**: Investigation and impact assessment
4. **Day 7-14**: Patch development and testing
5. **Day 14-30**: Release and public disclosure
6. **Day 30+**: Post-disclosure monitoring

## Security Contact

For security-related questions or concerns:

- **Security issues**: Use GitHub Security Advisories
- **General questions**: Create a GitHub Discussion
- **Urgent issues**: Contact maintainers directly

## Acknowledgments

We appreciate security researchers and users who help keep this project secure. Security contributors will be acknowledged in:

- **Security advisories**: Credit for discovery
- **CHANGELOG.md**: Recognition for contributions
- **Hall of Fame**: Public recognition (if desired)

Thank you for helping keep the Terraform Package Provider secure! 
