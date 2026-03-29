# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-03-29

### Added
- **Core Library**: Complete 80-bit IEEE 754 extended double precision floating-point implementation
- **Arithmetic Operations**: Full set of operations (add, subtract, multiply, divide, square root, natural logarithm, arctangent)
- **Comparison Operations**: Complete comparison suite (equal, less than, greater than, etc.)
- **Type Conversions**: Conversions to/from int32, int64, float32, float64
- **Exception Handling**: IEEE 754 compliant exception handling with customizable callbacks
- **Comprehensive Testing**: Unit tests with 48.2% code coverage
- **Performance Benchmarks**: Extensive benchmark suite for performance validation
- **Documentation**: Complete API reference, usage examples, and performance notes
- **CI/CD Pipeline**: GitHub Actions workflows for testing, linting, and releases
- **Development Tools**: Makefile with common development targets
- **Security Analysis**: CodeQL integration for automated security scanning
- **Dependency Management**: Automated dependency updates via Dependabot

### Features
- **Precision**: 80-bit extended precision with 64-bit mantissa
- **Compliance**: Full IEEE 754 standard implementation
- **Performance**: Optimized bit-level operations
- **Reliability**: Comprehensive error handling and edge case testing
- **Maintainability**: Well-documented code with professional structure

### Technical Details
- **Go Version**: Requires Go 1.22+
- **Architecture**: Cross-platform (Linux, macOS, Windows)
- **Testing**: Multi-version Go testing (1.21, 1.22, 1.23)
- **Coverage**: 48.2% test coverage with HTML reports
- **Linting**: Automated code quality checks (vet, golint, staticcheck)

### Infrastructure
- **GitHub Actions**: Complete CI/CD pipeline
- **Release Automation**: Automated GitHub releases on version tags
- **Documentation**: Hosted on pkg.go.dev
- **Coverage**: Integrated with Codecov
- **Security**: Weekly CodeQL security scans

---

## [0.1] - 2026-03-XX

### Added
- Initial implementation of 80-bit floating-point arithmetic
- Basic operations and type definitions
- Initial test suite
- Basic documentation

---

[1.0.0]: https://github.com/jenska/float/releases/tag/v1.0.0
[0.1]: https://github.com/jenska/float/releases/tag/0.1