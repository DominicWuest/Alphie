INSERT INTO todo.subscription (id, subscription_name, schedule, semester) VALUES (
    'Sem-02',
    '2nd Semester',
    '',
    ''
);

INSERT INTO todo.subscription (id, subscription_name, schedule, semester) VALUES (
    '401-0212-16L',
    'Analysis 1',
    '0 18 * * FRI',
    'F'
);

INSERT INTO todo.subscription (id, subscription_name, schedule, semester) VALUES (
    '252-0028-00L',
    'Digital Design and Computer Architecture',
    '0 0 * * TUE',
    'F'
);

INSERT INTO todo.subscription (id, subscription_name, schedule, semester) VALUES (
    '252-0029-00L',
    'Parallel Programming',
    '0 18 * * TUE',
    'F'
);

INSERT INTO todo.subscription (id, subscription_name, schedule, semester) VALUES (
    '252-0030-00L',
    'Algorithms and Probability',
    '0 0 * * THU',
    'F'
);

INSERT INTO todo.subscription_child (parent, child) VALUES (
    'Sem-02',
    '401-0212-16L'
);

INSERT INTO todo.subscription_child (parent, child) VALUES (
    'Sem-02',
    '252-0028-00L'
);

INSERT INTO todo.subscription_child (parent, child) VALUES (
    'Sem-02',
    '252-0029-00L'
);

INSERT INTO todo.subscription_child (parent, child) VALUES (
    'Sem-02',
    '252-0030-00L'
);