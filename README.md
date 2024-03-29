Elgato/Corsair Key Light control tool
=====================================

`keylightctl` is a CLI tool that allows you to control your Elgato/Corsair Key Light via the command line. It lets you set the brightness and the temperature without opening the Elgato Control Center app.

Installation
------------

To install `keylightctl` you need Go v1.20 or higher. Then run:

```bash
go install github.com/andrewslotin/keylightctl@latest
```

This will install the `keylightctl` binary into your `$GOPATH/bin` directory.

Usage
-----

`keylightctl` performs a discovery to find the first Key Light on your network. If you have multiple Key Lights, you can specify the IP addresses of those you want to control as arguments in following format: `<ip:port>`. The default port is `9123`.

The following command would set the brightness to 50% and the color temperature to 4500K:

```bash
keylightctl -b 50 -k 4500
```

Any of the flags can be omitted, in which case the corresponding setting will not be changed. For example, the following command would only set the brightness to 50% while leaving the color temperature unchanged:

```bash
keylightctl -b 50
```

To turn the light off, set the brightness to 0 by running:

```bash
keylightctl -b 0
```

To turn the light on, set the brightness to a value greater than 0.

To see all available flags, run:

```bash
keylightctl -h
```

License
-------

`keylightctl` is licensed under the MIT license. See the [LICENSE](LICENSE.md) file for details.
