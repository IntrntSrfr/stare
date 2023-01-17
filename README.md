# Discord logger for some events

## Requirements

- PostgreSQL database
- Golang
- Enough space :)

## How to run

idk yet im working on it.

A `config.json` file is required, make sure it's put wherever its being ran from.

| Key               | Value                        |
|-------------------|------------------------------|
| token             | Discord bot token            |
| connection_string | PostgreSQL connection string |
| owo_api_key       | OwO API Key                  |

## Structure?

- [`cmd/logger`](/cmd/logger)
  - Entrypoint
- [`/bot`](/bot)
  - Uses and combines all the other stuff in here
- [`/database`](/database)
  - Database to track guild configs
- [`/discord`](/discord)
  - Combines discord sessions into one unit with an event channel
- [`/kvstore`](/kvstore)
  - Persistent key-value store to track member and message data
