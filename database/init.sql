CREATE TABLE IF NOT EXISTS guilds (
	id             TEXT PRIMARY KEY,
	msg_edit_log   text,
	msg_delete_log text,
	ban_log        text,
	unban_log      text,
	join_log       text,
	leave_log      text
);