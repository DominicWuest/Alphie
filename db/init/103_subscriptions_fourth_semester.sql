INSERT INTO todo.subscription (id, subscription_name, schedule, semester) VALUES (
    'Sem-04',
    '4th Semester',
    '',
    ''
);

INSERT INTO todo.subscription (id, subscription_name, schedule, semester) VALUES (
    '252-0064-00L',
    'Computer Networks',
    '0 0 * * FRI',
    'F'
);

INSERT INTO todo.subscription (id, subscription_name, schedule, semester) VALUES (
    '252-0063-00L',
    'Data Modelling and Databases',
    '0 16 * * THU',
    'F'
);

INSERT INTO todo.subscription (id, subscription_name, schedule, semester) VALUES (
    '252-0058-00L',
    'Formal Methods and Function Programming',
    '0 0 * * MON',
    'F'
);

INSERT INTO todo.subscription (id, subscription_name, schedule, semester) VALUES (
    '401-0614-00L',
    'Probability and Statistics',
    '',
    ''
);

INSERT INTO todo.subscription (id, subscription_name, schedule, semester) VALUES (
    '401-0614-00L-0',
    'Exercise Sheet',
    '0 10 * * WED',
    'F'
);


INSERT INTO todo.subscription (id, subscription_name, schedule, semester) VALUES (
    '401-0614-00L-1',
    'Quiz',
    '0 10 * * WED',
    'F'
);

INSERT INTO todo.subscription_child (parent, child) VALUES (
    'Sem-04',
    '252-0064-00L'
);

INSERT INTO todo.subscription_child (parent, child) VALUES (
    'Sem-04',
    '252-0063-00L'
);

INSERT INTO todo.subscription_child (parent, child) VALUES (
    'Sem-04',
    '252-0058-00L'
);

INSERT INTO todo.subscription_child (parent, child) VALUES (
    'Sem-04',
    '401-0614-00L'
);

INSERT INTO todo.subscription_child (parent, child) VALUES (
    '401-0614-00L',
    '401-0614-00L-0'
);

INSERT INTO todo.subscription_child (parent, child) VALUES (
    '401-0614-00L',
    '401-0614-00L-1'
);