# Release notes

## v1.1.0
- Change license to BSD-2-Clause
- Follow REUSE specification, see https://reuse.software/
- Return payload even if NOK is set in header
- Update max app size to 128 kiB
- Reset read timeout to disabled in GetNameVersion
- Add function SetReadTimeoutNoErr, to use nicely with defer
- Update golanglint-ci
- Introduce use of safecast

Complete
[changelog](https://github.com/tillitis/tkeyclient/compare/v1.0.0...v1.1.0).


## v1.0.0
Going to version 1.0.0 to indicate that tkeyclient is stable and
production ready, according to Semantic Versioning.

- Bumping go dependencies.
- Removing stray dependency go-winres.
- Go lint using golangci-lint-action in CI.
- Updating README and copyright notice


## v0.0.8
Update dependencies. Updating the serial package to keep tkeyclient
buildable on darwin with go 1.21.

- go.bug.st/serial v1.6.1
- golang.org/x/crypto v0.13.0
- golang.org/x/sys v0.12.0


## v0.0.7

Just ripped from

https://github.com/tillitis/tillitis-key1-apps

No semantic changes.
