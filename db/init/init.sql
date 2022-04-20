CREATE SCHEMA todo;

CREATE TABLE todo.discord_user (
    id VARCHAR(19), -- discord snowflake ID's
    PRIMARY KEY(id)
);

CREATE TABLE todo.task (
  id  SERIAL NOT NULL,
  creator VARCHAR(19) references todo.discord_user (id) NOT NULL,
  title TEXT NOT NULL,
  description TEXT,
  PRIMARY KEY(id)
);

CREATE TABLE todo.completed (
    discord_user VARCHAR(19) references todo.discord_user (id) NOT NULL,
    task SERIAL references todo.task (id) NOT NULL,
    PRIMARY KEY (discord_user, task)
);
