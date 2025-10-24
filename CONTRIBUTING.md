# Contributing to Nexus

Thank you for your interest in contributing to Nexus! We welcome bug reports,
feature requests, documentation improvements, and code contributions.

## Before You Start

1. **Read the documentation**:
   - [Getting Started](docs/GETTING_STARTED.md) - Set up your development
     environment
   - [Development Guide](docs/DEVELOPMENT.md) - Learn the development workflow
   - [Architecture](docs/ARCHITECTURE.md) - Understand the system design

2. **Check existing issues**: Browse
   [GitHub Issues](https://github.com/retran/nexus/issues) to see if your idea
   or bug has already been reported

3. **Discuss large changes**: For significant features or architectural changes,
   open an issue first to discuss the approach

## Development Setup

See [docs/GETTING_STARTED.md](docs/GETTING_STARTED.md) for complete setup
instructions.

Quick start:

```bash
git clone https://github.com/retran/nexus.git
cd nexus
cp .env.example .env
task up
```

## Contribution Workflow

### 1. Fork and Branch

```bash
# Fork the repository on GitHub, then clone your fork
git clone https://github.com/YOUR_USERNAME/nexus.git
cd nexus

# Add upstream remote
git remote add upstream https://github.com/retran/nexus.git

# Create a feature branch
git checkout -b feature/your-feature-name
```

**Branch naming conventions**:

- `feature/` - New features
- `fix/` - Bug fixes
- `docs/` - Documentation updates
- `refactor/` - Code refactoring
- `test/` - Test improvements

Examples: `feature/user-authentication`, `fix/cors-headers`, `docs/api-guide`

### 2. Make Changes

Follow the patterns in [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) for
implementing changes:

- **Backend**: Use code generation (sqlc, gqlgen, genqlient)
- **Frontend**: Follow React/TypeScript best practices
- **Database**: Edit `postgres/schema.hcl`, generate migrations

**Before committing**:

```bash
# Format code
task backend:format
task frontend:format

# Run linters
task backend:lint
task frontend:lint

# Run tests
task backend:test
task frontend:test
```

### 3. Commit Your Changes

We follow [Conventional Commits](https://www.conventionalcommits.org/)
specification:

**Format**: `<type>(<scope>): <description>`

**Types**:

- `feat` - New feature
- `fix` - Bug fix
- `docs` - Documentation only
- `style` - Code style changes (formatting, no logic change)
- `refactor` - Code refactoring
- `perf` - Performance improvements
- `test` - Adding or updating tests
- `chore` - Build process, dependencies, tooling

**Examples**:

```bash
git commit -m "feat(api): add user profile endpoint"
git commit -m "fix(ui): correct header alignment on mobile"
git commit -m "docs: update development guide"
git commit -m "refactor(db): extract user validation logic"
git commit -m "test(api): add tests for authentication flow"
git commit -m "chore(deps): update Go dependencies"
```

**Scope** (optional) can be:

- `api` - Backend API changes
- `ui` - Frontend changes
- `db` - Database changes
- `worker` - Temporal worker
- `infra` - Infrastructure/Docker

### 4. Push and Create Pull Request

```bash
# Push to your fork
git push origin feature/your-feature-name
```

Then on GitHub:

1. Navigate to your fork
2. Click "Pull Request"
3. Select your feature branch
4. Fill out the PR template with:
   - Description of changes
   - Related issue numbers (if any)
   - Screenshots (for UI changes)
   - Testing steps

### 5. Code Review

- Respond to review comments promptly
- Make requested changes in new commits
- Push updates to the same branch
- Once approved, maintainers will merge your PR

## Pull Request Guidelines

### PR Title

Use the same format as commit messages:

```
feat(api): add user profile endpoint
fix(ui): correct mobile responsiveness
docs: improve getting started guide
```

### PR Description

Include:

- **What**: What does this PR do?
- **Why**: Why is this change needed?
- **How**: How does it work?
- **Testing**: How was it tested?
- **Screenshots**: For UI changes
- **Breaking Changes**: List any breaking changes

Example:

```markdown
## What

Adds user profile API endpoint with avatar upload support.

## Why

Users need to be able to update their profile information and upload avatars.

## How

- Created `UpdateProfile` mutation in GraphQL schema
- Added S3 upload handler for avatars
- Updated database schema with `avatar_url` column

## Testing

- Unit tests added for profile update logic
- Manual testing with Postman
- Tested avatar upload with 5MB file

## Breaking Changes

None
```

### Checklist

Before submitting, ensure:

- [ ] Code follows project style guidelines
- [ ] Tests added/updated for new functionality
- [ ] Documentation updated (if needed)
- [ ] All tests pass locally
- [ ] No linting errors
- [ ] Commit messages follow convention
- [ ] PR title follows convention

## Code Quality Standards

### Go (Backend)

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting (automatic via `task backend:format`)
- Run `golangci-lint` (automatic via `task backend:lint`)
- Write table-driven tests
- Add comments for exported functions
- Handle errors explicitly

### TypeScript (Frontend)

- Follow TypeScript strict mode
- Use functional components with hooks
- Extract reusable logic into custom hooks
- Use TypeScript interfaces/types (avoid `any`)
- Write unit tests for complex logic
- Follow Refine.dev patterns

### SQL (Database)

- Use lowercase keywords
- Use `snake_case` for tables and columns
- Add indexes for foreign keys and frequently queried columns
- Write migrations in `postgres/schema.hcl` (not SQL directly)
- Add comments for complex queries

## Testing Requirements

### Unit Tests

- Required for new features
- Maintain >80% code coverage
- Test happy path and error cases
- Use mocks for external dependencies

### Integration Tests

- Test API endpoints end-to-end
- Verify database interactions
- Test authentication/authorization

### Manual Testing

For UI changes:

- Test on Chrome, Firefox, Safari
- Test responsive design (mobile, tablet, desktop)
- Verify accessibility (keyboard navigation, screen readers)

## Reporting Bugs

When reporting bugs, include:

1. **Description**: What happened vs. what you expected
2. **Steps to reproduce**:

   ```
   1. Go to http://nexus.local
   2. Click "Login"
   3. See error...
   ```

3. **Environment**:
   - OS: macOS 14.2
   - Docker: 24.0.7
   - Browser: Chrome 120 (for UI bugs)
4. **Logs**: Relevant error messages or stack traces
5. **Screenshots**: For visual bugs

## Suggesting Features

When suggesting features, include:

1. **Problem**: What problem does this solve?
2. **Proposed solution**: How should it work?
3. **Alternatives**: Other approaches considered?
4. **Implementation notes**: Any technical considerations?

## Documentation

Documentation is as important as code! When updating docs:

- Use clear, concise language
- Include code examples
- Add links to related docs
- Update table of contents (if applicable)

## License

By contributing, you agree that your contributions will be licensed under the
Apache-2.0 License. See [LICENSE](LICENSE).

## Questions?

- Open an issue for questions
- Check [docs/](docs/) for detailed guides
- Review existing code for patterns

Thank you for contributing to Nexus! 🚀
