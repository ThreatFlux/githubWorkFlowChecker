# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ----------------- |
| 1.0.x   | :white_check_mark: |

## Security Requirements

### GitHub Token Permissions
The GitHub token used with this tool requires the following permissions:
- `contents: write` for creating commits
- `pull_requests: write` for creating pull requests
- Access to workflow files

### Rate Limiting
- Implements GitHub API rate limit handling
- Uses exponential backoff for retries
- Transparent handling of rate limit errors
- Respects GitHub's secondary rate limits

### File Access
- Scans only `.github/workflows/*.{yml,yaml}` files
- Implements safe path resolution
- Prevents path traversal attacks
- Enforces file size and content limits

## Reporting a Vulnerability

If you discover a security vulnerability in this project:

1. **Do Not** open a public issue
2. Email wyattroersma@gmail.com with the subject "Security Vulnerability: githubWorkFlowChecker"
3. Include:
   - Clear description of the vulnerability
   - Steps to reproduce
   - Potential impact assessment
   - Suggested remediation (if available)

Response time: Within 48 hours
- Security advisory will be created if confirmed
- Fix will be prioritized based on severity
- CVE will be requested if applicable

## Security Best Practices

### Token Management
1. Use GitHub repository secrets for tokens
2. Never commit tokens to source control
3. Use minimal required permissions
4. Implement automatic token rotation
5. Monitor token usage

### Action Configuration
1. Pin actions to full-length commit SHAs
2. Validate workflow file syntax
3. Keep dependencies updated
4. Monitor GitHub Security Advisories

### Error Handling
1. Sanitize error messages
2. Implement proper logging
3. Clean up resources on failure
4. Handle API errors gracefully

## Security Features

### Input Validation
- Safe YAML parsing using gopkg.in/yaml.v3
- Strict action reference validation
- Version string format verification
- Secure path resolution

### Docker Security
- Non-root user execution
- Minimal base image (Alpine)
- Regular security updates
- Signed container images
- Generated SBOM
- Dropped capabilities
- No-new-privileges enforcement

### Network Security
- HTTPS-only API communication
- Request timeouts
- Response validation
- Rate limit compliance

## Incident Response

Security incident handling process:

1. Immediate assessment upon report
2. Investigation and containment
3. User notification if required
4. Patch development and testing
5. Security advisory publication
6. Post-incident analysis

## Security Updates

Security patches are delivered via:
1. Regular releases
2. Emergency patches for critical issues
3. GitHub Security Advisories
4. Automated dependency updates

## Contact

- Security Contact: wyattroersma@gmail.com
- Response Time: 48 hours
- Repository: https://github.com/ThreatFlux/githubWorkFlowChecker

## Security Measures

This tool implements several security measures:

1. Supply Chain Security:
   - Pinned dependency versions
   - Regularly updated dependencies
   - SBOM generation
   - Container signing

2. Runtime Security:
   - Non-root user execution
   - Minimal container permissions
   - Secure file handling
   - Resource cleanup

3. API Security:
   - Token validation
   - Permission verification
   - Secure communication

4. Code Security:
   - Regular security scans
   - Dependency vulnerability checks
   - Static analysis
   - Security-focused code review