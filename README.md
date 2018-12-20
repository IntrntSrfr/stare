# Discord logger for some events
### Made for single server use as of now.

Steps: 
1. Create config.json file from config-template.json template
2. run `go build -o logger.exe` inside this folder
3. third

Create a config.json file


| Key           | Value                                |
| ------------- |--------------------------------------|
| Token         | Discord bot token                    |
| PBToken       | Pastebin token                       |
| MsgEdit       | Edited messages log channel ID       |
| MsgDelete     | Deleted messages log channel ID      |
| Ban           | Ban log channel ID                   |
| Unban         | Unban messages log channel ID        |
| Join          | Join messages log channel ID         |
| Leave         | Leave messages log channel ID        |