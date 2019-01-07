
CREATE TABLE IF NOT EXISTS discordguilds
(
    uid serial NOT NULL,
    guildid character varying(30) NOT NULL,
    msgeditlog character varying(30) NOT NULL,
    msgdeletelog character varying(30) NOT NULL,
    banlog character varying(30) NOT NULL,
    unbanlog character varying(30) NOT NULL,
    joinlog character varying(30) NOT NULL,
    leavelog character varying(30) NOT NULL,
    CONSTRAINT discordguild_pkey PRIMARY KEY (uid)
)
WITH (OIDS=FALSE); 