# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ----------------- |
| 1.0.x   | :white_check_mark: |

## Security Requirements

### GitHub Token Permissions
The GitHub token used with this tool requires the following permissions:
- `repo` scope for repository access and PR creation
- `workflow` scope for workflow file access

### Rate Limiting
- The tool respects GitHub API rate limits
- Default backoff strategy is implemented
- Rate limit errors are handled gracefully

### File Access
- Only YAML files within the `.github/workflows` directory are processed
- Absolute paths are not allowed
- Path traversal attempts are blocked
- File size limits are enforced

## Reporting a Vulnerability

If you discover a security vulnerability in this project:

1. **Do Not** open a public issue
2. Email your findings to security@threatflux.com
3. Include:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)

You should receive a response within 48 hours. If the issue is confirmed:
1. A security advisory will be created
2. The fix will be implemented
3. A new version will be released
4. CVE will be requested if applicable

## Security Best Practices

### Token Management
1. Use environment variables for token storage
2. Never commit tokens to source control
3. Use repository secrets for CI/CD workflows
4. Rotate tokens regularly
5. Use tokens with minimal required permissions

### Configuration
1. Always validate workflow files before processing
2. Use explicit version references in workflows
3. Keep dependencies up to date
4. Monitor security advisories for dependencies

### Error Handling
1. Avoid exposing sensitive information in error messages
2. Log security-relevant events
3. Implement proper cleanup on failures
4. Handle API errors gracefully

## Security Features

### Input Validation
- YAML parsing using safe yaml.v3 package
- Action reference format validation
- Version string validation
- Path validation

### Authentication
- OAuth2 token validation
- Automatic token refresh
- Scope validation
- Repository permission checks

### Network Security
- HTTPS-only communication
- Request timeouts
- Response validation
- Rate limit handling

## Incident Response

In case of a security incident:

1. The security team will be notified immediately
2. The incident will be investigated
3. Affected users will be notified
4. A post-mortem will be conducted
5. Security measures will be updated as needed

## Security Updates

Security updates are delivered through:
1. Regular releases with bug fixes
2. Emergency patches for critical vulnerabilities
3. Security advisories in GitHub

## Contact

- Security Email: security@threatflux.com
- PGP Key: [security-pgp.asc](https://threatflux.com/security-pgp.asc)
- Response Time: 48 hours
- Emergency Contact: security-emergency@threatflux.com
