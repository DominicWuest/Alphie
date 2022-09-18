CREATE SCHEMA lecture_clippers;

CREATE TABLE lecture_clippers.clippers (
    id VARCHAR(20) NOT NULL,
    semester CHAR NOT NULL,
    CHECK (semester IN ('F', 'H', 'B', 'N')), -- Spring, Fall, Both, None
    PRIMARY KEY (id)
);

CREATE TABLE lecture_clippers.schedule (
    id VARCHAR(20) NOT NULL,
    room_url VARCHAR(15) NOT NULL,
    schedule VARCHAR(64) NOT NULL, -- In the cronjob format
    duration_minutes INT NOT NULL, -- Duration of the lecture in minutes
    PRIMARY KEY (id, schedule),
    FOREIGN KEY (id) REFERENCES lecture_clippers.clippers(id)
);

CREATE TABLE lecture_clippers.lecture_alias (
    id VARCHAR(20) NOT NULL,
    aliases VARCHAR(64)[],
    PRIMARY KEY (id),
    FOREIGN KEY (id) REFERENCES lecture_clippers.clippers(id)
)