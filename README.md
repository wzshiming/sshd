# sshd

ssh server

[![Build](https://github.com/wzshiming/sshd/actions/workflows/go-cross-build.yml/badge.svg)](https://github.com/wzshiming/sshd/actions/workflows/go-cross-build.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/wzshiming/sshd)](https://goreportcard.com/report/github.com/wzshiming/sshd)
[![GoDoc](https://godoc.org/github.com/wzshiming/sshd?status.svg)](https://godoc.org/github.com/wzshiming/sshd)
[![GitHub license](https://img.shields.io/github/license/wzshiming/sshd.svg)](https://github.com/wzshiming/sshd/blob/master/LICENSE)

This project is to add protocol support for the [sshproxy](https://github.com/wzshiming/sshproxy), or it can be used alone

## Usage

[API Documentation](https://godoc.org/github.com/wzshiming/sshd)

[Example](https://github.com/wzshiming/sshd/blob/master/cmd/sshd/main.go)

- [x] Support for the Direct TCP IP command
- [x] Support for the TCP IP Forward command
- [x] Support for the Direct Stream Local command
- [x] Support for the Stream Local Forward command
- [x] Support for the Session command
  - [x] env
  - [x] exec
  - [ ] shell
  - [ ] subsystem

## License

Licensed under the MIT License. See [LICENSE](https://github.com/wzshiming/sshd/blob/master/LICENSE) for the full license text.
