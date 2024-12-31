# Stare - A Discord server logger

## Requirements

- Go 1.22

## How to run

A `config.json` file is required, make sure it's put wherever its being run from. 
Use the [template json file](cmd/logger/config-template.json) as a starting point.

```bash
$ cd cmd/logger
$ go build
$ ./logger
```

## What gets logged:

- When a user joins the server
- When a user leaves the server
- When a message is deleted
- When messages are bulk deleted
- When a message is edited
- When a user is banned
- When a user is unbanned

## Commands

- /help
- /info
- /settings set
  - Set channels to post logs for events 
- /settings view
