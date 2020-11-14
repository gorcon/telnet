# Changelog
All notable changes to this project will be documented in this file.

**ATTN**: This project uses [semantic versioning](http://semver.org/).

## [Unreleased]

## [v1.0.1] - 2020-11-14
### Added
- Added the ability to run the help command on a real 7 Days to Die server. To do this, set environment variables 
`TEST_7DTD_SERVER=true`, `TEST_7DTD_SERVER_ADDR` and `TEST_7DTD_SERVER_PASSWORD` with address and password from 
7 Days to Die remote console.  

### Changed
- Changed CI workflows and related badges. Integration with Travis-CI was changed to GitHub actions workflow. Golangci-lint 
job was joined with tests workflow.  

## v1.0.0 - 2020-10-06
### Added
- Initial implementation.

[Unreleased]: https://github.com/gorcon/telnet/compare/v1.0.1...HEAD
[v1.0.1]: https://github.com/gorcon/telnet/compare/v1.0.0...v1.0.1
