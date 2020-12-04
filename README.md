# discordkvs
> Discord storage library for bots

[![PkgGoDev](https://pkg.go.dev/badge/github.com/ethanent/discordkvs)](https://pkg.go.dev/github.com/ethanent/discordkvs)

## Install

```sh
go get github.com/ethanent/discordkvs
```

## Features

- Store key-value pairs within a Discord channel in a server, rather than storing data in your own database.
- Cleans up data periodically, based on random chance for performance.
- Can store data larger than a Discord message by intelligently switching to attachment files rather than inline text. (Higher latency, but can store larger values.)
- Encrypt data, so if you keep your secure KVS Application ID a secret (eg. by keeping codebase private), the data should be reasonably difficult for others to access.
    - Encryption fails early, so there should generally not be encryption errors after Application is initialized.
- Optionally, Application may be allowed to read data from other bots / users sharing the same Application ID.

## Usage

```go
// Assuming you already have a *discordgo.Session called s...

app, err = discordkvs.NewApplication(s, "MyApp-ID891173")

if err != nil {/* handle error, probably panic during startup */}

// Once bot is open, you can begin saving / reading data.

err = app.Set("732134812499836941", "myKey", []byte("myValue"))

// Read value for myKey in guild 732134812499836941:

data, err := app.Get("732134812499836941", "myKey")
```