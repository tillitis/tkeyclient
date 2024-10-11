[![ci](https://github.com/tillitis/tkeyclient/actions/workflows/ci.yaml/badge.svg?branch=main&event=push)](https://github.com/tillitis/tkeyclient/actions/workflows/ci.yaml) [![Go Reference](https://pkg.go.dev/badge/github.com/tillitis/tkeyclient.svg)](https://pkg.go.dev/github.com/tillitis/tkeyclient)

# Tillitis TKey Client package

A Go package for controlling a [Tillitis](https://tillitis.se/) TKey,
upload device apps, and communicate with it.

See the [Go doc](https://pkg.go.dev/github.com/tillitis/tkeyclient)
for `tkeyclient` for details on how to call the functions.

See [tkey-ssh-agent](https://github.com/tillitis/tkey-ssh-agent) or
[tkeysign](https://github.com/tillitis/tkeysign) for
example applications using this Go package.

Release notes in [RELEASE.md](RELEASE.md).

## Licenses and SPDX tags

Unless otherwise noted, the project sources are copyright Tillitis AB,
licensed under the terms and conditions of the "BSD-2-Clause" license.
See [LICENSE](LICENSE) for the full license text.

Until Oct 7, 2024, the license was GPL-2.0 Only.

External source code we have imported are isolated in their own
directories. They may be released under other licenses. This is noted
with a similar `LICENSE` file in every directory containing imported
sources.

The project uses single-line references to Unique License Identifiers
as defined by the Linux Foundation's [SPDX project](https://spdx.org/)
on its own source files, but not necessarily imported files. The line
in each individual source file identifies the license applicable to
that file.

The current set of valid, predefined SPDX identifiers can be found on
the SPDX License List at:

https://spdx.org/licenses/

We attempt to follow the [REUSE
specification](https://reuse.software/).
