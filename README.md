# discordkvs
> Discord storage library for bots

[![PkgGoDev](https://pkg.go.dev/badge/github.com/ethanent/discordkvs)](https://pkg.go.dev/github.com/ethanent/discordkvs)

## Install

```sh
go get github.com/ethanent/discordkvs
```

## Features

- Store key-value pairs within a Discord channel in a server, rather than storing data in your own database.
- Encrypt data, so if you keep your secure KVS Application ID a secret (eg. by keeping codebase private), the data should not be accessible to others.
    - Encryption fails early, so there should generally not be encryption errors after Application is initialized.
- Optionally, Application may be allowed to read data from other bots sharing the same Application ID.
