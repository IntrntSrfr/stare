# Discord logger for some events
### Made for single server use as of now.

Tutorial: 
1. Create config.json file from config-template.json template
2. Run `go build -o logger.exe` inside this folder
3. Run `logger.exe`
4. Congratulations you did it

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