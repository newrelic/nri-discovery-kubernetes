# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/).

Unreleased section should follow [Release Toolkit](https://github.com/newrelic/release-toolkit#render-markdown-and-update-markdown)

## Unreleased

## v1.13.5 - 2025-11-03

### ⛓️ Dependencies
- Updated golang version to v1.25.3

## v1.13.4 - 2025-09-15

### ⛓️ Dependencies
- Updated github.com/spf13/viper to v1.21.0 - [Changelog 🔗](https://github.com/spf13/viper/releases/tag/v1.21.0)

## v1.13.3 - 2025-09-08

### ⛓️ Dependencies
- Updated github.com/spf13/pflag to v1.0.10 - [Changelog 🔗](https://github.com/spf13/pflag/releases/tag/v1.0.10)

## v1.13.2 - 2025-09-01

### ⛓️ Dependencies
- Updated github.com/spf13/pflag to v1.0.9 - [Changelog 🔗](https://github.com/spf13/pflag/releases/tag/v1.0.9)

## v1.13.1 - 2025-08-28

### ⛓️ Dependencies
- Updated golang patch version to v1.24.6

## v1.13.0 - 2025-07-21

### ⛓️ Dependencies
- Updated github.com/spf13/pflag to v1.0.7 - [Changelog 🔗](https://github.com/spf13/pflag/releases/tag/v1.0.7)
- Updated golang patch version to v1.24.5
- Upgraded golang.org/x/oauth2 from 0.25.0 to 0.27.0

## v1.12.1 - 2025-07-10

### ⛓️ Dependencies
- Updated github.com/go-viper/mapstructure/v2

## v1.12.0 - 2025-07-10

### ⛓️ Dependencies
- Upgraded golang.org/x/net from 0.33.0 to 0.38.0

## v1.11.3 - 2025-06-30

### ⛓️ Dependencies
- Updated golang version to v1.24.4

## v1.11.2 - 2025-03-31

### ⛓️ Dependencies
- Updated github.com/spf13/viper to v1.20.1 - [Changelog 🔗](https://github.com/spf13/viper/releases/tag/v1.20.1)

## v1.11.1 - 2025-03-17

### ⛓️ Dependencies
- Updated github.com/spf13/viper to v1.20.0 - [Changelog 🔗](https://github.com/spf13/viper/releases/tag/v1.20.0)

## v1.11.0 - 2025-03-10

### 🚀 Enhancements
- Add FIPS compliant packages

## v1.10.1 - 2025-02-03

### ⛓️ Dependencies
- Updated github.com/spf13/pflag to v1.0.6 - [Changelog 🔗](https://github.com/spf13/pflag/releases/tag/v1.0.6)
- Updated golang patch version to v1.23.5

## v1.10.0 - 2024-12-19

### 🚀 Enhancements
- Updated golang.org/x/net to v0.33.0

## v1.9.4 - 2024-12-16

### ⛓️ Dependencies
- Updated golang patch version to v1.23.4

## v1.9.3 - 2024-09-16

### ⛓️ Dependencies
- Updated golang version to v1.23.1
- Updated kubernetes packages to v0.31.1
- Updated github.com/spf13/viper to v1.19.0 - [Changelog 🔗](https://github.com/spf13/viper/releases/tag/v1.19.0)

## v1.9.2 - 2024-07-08

### ⛓️ Dependencies
- Updated golang version to v1.22.5

## v1.9.1 - 2024-05-13

### ⛓️ Dependencies
- Updated golang version to v1.22.3

## v1.9.0 - 2024-05-06

### ⛓️ Dependencies
- Updated golang version to v1.22.2
- Updated kubernetes packages to v0.30.0
- Upgraded golang.org/x/net from 0.19.0 to 0.23.0

## v1.8.0 - 2024-04-15

### 🛡️ Security notices
- Updated dependencies

### ⛓️ Dependencies
- Upgraded google.golang.org/protobuf from 1.31.0 to 1.33.0

## 1.6.2
### Changed
- Bump go version and dependencies

## 1.6.1
### Changed
- Disable CGO

## 1.6.0
### Changed
- Upgrade Go to 1.19 and bump dependencies

## 1.4.2
### Changed
- Bump go version and dependencies

## 1.4.1
### Changed
- Bump go version and dependencies

## 1.4.0
### Changed
- Update Kubernetes Go dependencies to latest versions

## 1.3.1
### Changed
- CI/CD pipeline migrated to GitHub Actions

## 1.3.0
### Changelog

- Docs update
- Check if command line args were provided
- Add Open Source Policy Workflow (#11)
- Close request body
- Added auto-detection for kubelet client config by using --auto_config cmd line arg
- b226a2f trigger pipeline
- Update linter version


## 1.2.0
### Changelog

- Filter non-running containers
- Update gcp.yaml.template
- Update minikube.yaml
- Fixed failing test

## 1.1.0
### Changed
   - Optional `insecure` flag has been deprecated in favor of `tls`
     TLS connections are now disabled by default. If you want to use SSL, use `tls` flag (or set `insecure` to false)
- Optional `port` flag default value has been changed to `10255`

## 1.0.1
### Added
- Support for returning container ports as per the deployment spec.
  The results include the "index" of the port as well as the name if available
- Support for using the node name when querying the Kubelet for containers.
  Fixes issue with deployments with hostNetwork=false (the default)
## 1.0.0
### Added
- Initial version
