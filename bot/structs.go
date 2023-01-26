package bot

const (
	Red    = 0xC80000
	Orange = 0xF08152
	Blue   = 0x61D1ED
	Green  = 0x00C800
	White  = 0xFFFFFF
)

type LogType int

const (
	MessageDeleteType LogType = 1 << iota
	MessageDeleteBulkType
	MessageUpdateType
	GuildJoinType
	GuildLeaveType
	GuildBanType
	GuildUnbanType
)
