# Change Log

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/).

## 1.0.1
### Added
- Support for returning container ports as per the deployment spec.
  The results include the "index" of the port as well as the name if available
- Support for using the node name when querying the Kubelet for containers.
  Fixes issue with deployments with hostNetwork=false (the default)
## 1.0.0
### Added
- Initial version
