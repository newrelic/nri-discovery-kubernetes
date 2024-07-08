# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/).

Unreleased section should follow [Release Toolkit](https://github.com/newrelic/release-toolkit#render-markdown-and-update-markdown)

## Unreleased

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
