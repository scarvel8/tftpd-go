# tftpd-go
TFTPd server implementation written in Go, based off RFC 1350

BUILDING
========

go build tftpd-go.go

USAGE
=====

./tftpd-go.go

(requires superuser access as the application needs to bind to a privileged port)

This is my own implementation of a TFTPd server based off RFC 1350 as an exercise for myself to get better acquainted with golang.

TODO:

* Implement write capabilities
* Better error checking and reporting
* BASH script for building / installing and adding as system service
* Implement goroutines for concurrent connections
* Variable optimizations
* Implement a state machine
* Email support
* Specify timeouts
* Enforce chroot
* Verbose mode
