CREATE SCHEMA todo;

CREATE TABLE todo.discord_user (
    id VARCHAR(19), -- discord snowflake ID's
    PRIMARY KEY(id)
);

INSERT INTO todo.discord_user (id) VALUES ('0'); -- The user id representing the bot itself

CREATE TABLE todo.task (
  id  SERIAL NOT NULL,
  creator VARCHAR(19) REFERENCES todo.discord_user (id) NOT NULL,
  title TEXT NOT NULL,
  description TEXT NOT NULL,
  PRIMARY KEY(id)
);

CREATE TABLE todo.active (
    discord_user VARCHAR(19) REFERENCES todo.discord_user (id) NOT NULL,
    task SERIAL REFERENCES todo.task (id) NOT NULL,
    PRIMARY KEY (discord_user, task)
);

CREATE TABLE todo.archived (
    discord_user VARCHAR(19) REFERENCES todo.discord_user (id) NOT NULL,
    task SERIAL REFERENCES todo.task (id) NOT NULL,
    PRIMARY KEY (discord_user, task)
);

CREATE TABLE todo.completed (
    discord_user VARCHAR(19) REFERENCES todo.discord_user (id) NOT NULL,
    task SERIAL REFERENCES todo.task (id) NOT NULL,
    PRIMARY KEY (discord_user, task)
);

CREATE TABLE todo.subscription (
    id VARCHAR(20) NOT NULL,
    subscription_name VARCHAR(128) NOT NULL,
    schedule VARCHAR(64) NOT NULL,
    PRIMARY KEY (id)
);

CREATE TABLE todo.subscription_child (
    parent VARCHAR(20) NOT NULL REFERENCES todo.subscription (id),
    child VARCHAR(20) NOT NULL REFERENCES todo.subscription (id),
    CHECK (parent <> child),
    PRIMARY KEY (parent, child)
);

CREATE TABLE todo.subscribed_to (
    discord_user VARCHAR(19) REFERENCES todo.discord_user (id) NOT NULL,
    subscription VARCHAR(20) REFERENCES todo.subscription (id) NOT NULL,
    PRIMARY KEY (discord_user, subscription)
);