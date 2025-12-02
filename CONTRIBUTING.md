# Contributing to EasyMark

Thank you for your interest in contributing to EasyMark! This document provides guidelines and information for contributors.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Making Changes](#making-changes)
- [Pull Request Process](#pull-request-process)
- [Style Guidelines](#style-guidelines)

## Code of Conduct

Please be respectful and constructive in all interactions. We are committed to providing a welcoming and inclusive environment for everyone.

## Getting Started

### Prerequisites

- Node.js 18+
- Go 1.21+
- Python 3.10+ (for plugin development)
- Git

### Fork and Clone

1. Fork the repository on GitHub
2. Clone your fork locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/EasyMark.git
   cd easymark
   ```
3. Add upstream remote:
   ```bash
   git remote add upstream https://github.com/aaaazbwzbw/EasyMark.git
   ```

## Development Setup

### Install Dependencies

```bash
# Frontend
cd frontend && npm install

# Electron
cd ../host-electron && npm install

# Backend
cd ../backend-go && go mod download
```

### Run Development Environment

```bash
# Terminal 1: Backend
cd backend-go && go run .

# Terminal 2: Frontend
cd frontend && npm run dev

# Terminal 3: Electron
cd host-electron && npm run dev
```

## Making Changes

### Branch Naming

- `feature/` - New features
- `fix/` - Bug fixes
- `docs/` - Documentation updates
- `refactor/` - Code refactoring

Example: `feature/add-video-annotation`

### Commit Messages

Follow conventional commits format:

```
type(scope): description

[optional body]

[optional footer]
```

Types:
- `feat` - New feature
- `fix` - Bug fix
- `docs` - Documentation
- `style` - Formatting
- `refactor` - Code refactoring
- `test` - Adding tests
- `chore` - Maintenance

Example:
```
feat(annotation): add polygon smoothing tool

- Added bezier curve smoothing for polygon vertices
- Added UI toggle in toolbar
```

## Pull Request Process

1. **Update your fork**
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature
   ```

3. **Make your changes**
   - Write clean, documented code
   - Add tests if applicable
   - Update documentation if needed

4. **Test your changes**
   ```bash
   # Run frontend tests
   cd frontend && npm run test

   # Run backend tests
   cd backend-go && go test ./...
   ```

5. **Submit Pull Request**
   - Fill out the PR template
   - Link any related issues
   - Request review from maintainers

## Style Guidelines

### TypeScript/Vue

- Use TypeScript strict mode
- Follow Vue 3 Composition API patterns
- Use `<script setup>` syntax
- Format with Prettier

### Go

- Follow standard Go conventions
- Use `gofmt` for formatting
- Write descriptive error messages

### Python (Plugins)

- Follow PEP 8
- Use type hints
- Document public functions

### General

- Keep functions small and focused
- Write self-documenting code
- Add comments for complex logic
- Internationalize all user-facing strings

## Questions?

Feel free to open an issue for any questions or discussions!

---

Thank you for contributing to EasyMark!
