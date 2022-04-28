INSERT INTO todo.subscription (id, subscription_name, schedule, semester) VALUES (
    'Sem-04',
    '4th Semester',
    '',
    ''
);

INSERT INTO todo.subscription (id, subscription_name, schedule, semester) VALUES (
    '252-0064-00L',
    'Computer Networks',
    '@every 1m', -- temporary for testing
    'F'
);

INSERT INTO todo.subscription (id, subscription_name, schedule, semester) VALUES (
    '252-0063-00L',
    'Data Modelling and Databases',
    '@every 1m', -- temporary for testing
    'F'
);

INSERT INTO todo.subscription (id, subscription_name, schedule, semester) VALUES (
    '252-0058-00L',
    'Formal Methods and Function Programming',
    '@every 1m', -- temporary for testing
    'F'
);

INSERT INTO todo.subscription (id, subscription_name, schedule, semester) VALUES (
    '401-0614-00L',
    'Probability and Statistics',
    '@every 1m', -- temporary for testing
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