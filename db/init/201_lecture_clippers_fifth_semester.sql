-- Computer Systems
INSERT INTO lecture_clippers.clippers VALUES
    ('252-0217-00', 'H');

INSERT INTO lecture_clippers.schedule VALUES
    ('252-0217-00', 'cab-g-61', '0 10 * * MON', 120),
    ('252-0217-00', 'cab-g-61', '0 10 * * FRI', 120);

INSERT INTO lecture_clippers.lecture_alias VALUES
    ('252-0217-00', '{"computer systems", "compsys"}');