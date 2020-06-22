# tftp-go

TFTP server implementation in Go.

## Usage

For an example server implementation, see [the example directory](./example).

This server provides read/write access to `$PWD`.

To run the server (from the `example` directory):

```
$ go build main.go
$ echo "Hello world!" > file
$ sudo ./main
```

To access the server with curl (for example):

```
$ curl tftp://localhost/file
Hello world!
```

## Notes

A client implementation could easily be built on top since the packet
serialization/deserialization is in place and not tied to either the server or
the client side. If you're looking to build this feature, let us know through
an issue on GitHub, or directly with a pull request for this functionality.

## RFCs

Other RFCs are informational or obsoleted by newer versions.

* [1350](https://tools.ietf.org/html/rfc1350): THE TFTP PROTOCOL (REVISION 2)
* [2347](https://tools.ietf.org/html/rfc2347): TFTP Option Extension
* [2348](https://tools.ietf.org/html/rfc2348): TFTP Blocksize Option
* [2349](https://tools.ietf.org/html/rfc2349): TFTP Timeout Interval and Transfer Size Options

## License

This project is available under the [Apache 2.0](./LICENSE) license.
