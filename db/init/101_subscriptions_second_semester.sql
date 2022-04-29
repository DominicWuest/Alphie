INSERT INTO todo.subscription (id, subscription_name, schedule, semester) VALUES (
    'Sem-02',
    '2nd Semester',
    '',
    ''
);

INSERT INTO todo.subscription (id, subscription_name, schedule, semester) VALUES (
    '401-0212-16L',
    'Analysis 1',
    '@every 1m', -- temporary for testing
    'F'
);

INSERT INTO todo.subscription (id, subscription_name, schedule, semester) VALUES (
    '252-0028-00L',
    'Digital Design and Computer Architecture',
    '@every 1m', -- temporary for testing
    'F'
);

INSERT INTO todo.subscription (id, subscription_name, schedule, semester) VALUES (
    '252-0029-00L',
    'Parallel Programming',
    '@every 1m', -- temporary for testing
    'F'
);

INSERT INTO todo.subscription (id, subscription_name, schedule, semester) VALUES (
    '252-0030-00L',
    'Algorithms and Probability',
    '@every 1m', -- temporary for testing
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