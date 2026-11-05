CREATE TABLE IF NOT EXISTS World (
    id integer NOT NULL,
    randomNumber integer NOT NULL,
    PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS Fortune (
    id integer NOT NULL,
    message varchar(2048) NOT NULL,
    PRIMARY KEY (id)
);

TRUNCATE World;
INSERT INTO World (id, randomNumber)
SELECT generate_series(1, 10000), floor(random() * 10000 + 1)::int;

TRUNCATE Fortune;
INSERT INTO Fortune (id, message) VALUES
    (1, 'fortune: No such file or directory'),
    (2, 'A computer scientist is someone who fixes things that aren''t broken.'),
    (3, 'After enough decimal places, nobody gives a damn.'),
    (4, 'A bad random number generator: 1, 1, 1, 1, 1, 4.33e+67, 1, 1, 1'),
    (5, 'A computer program does what you tell it to do, not what you want it to do.'),
    (6, 'Emacs is a nice operating system, but it lacks a good editor. I''m using Vim.'),
    (7, 'How many Microsoft engineers does it take to change a light bulb? None, because they simply declare darkness as the new standard.'),
    (8, 'The generation of random numbers is too important to be left to chance.'),
    (9, 'A primary reason IBM''s 360 architecture lasted so long is because of the jump instruction format? Bah, it was because of RPG.'),
    (10, 'Unix is user-friendly. It''s just picky about who its friends are.'),
    (11, 'Computers are like air conditioners: they stop working if you open Windows.'),
    (12, 'The best performance improvement is the transition from working to not-working state.');
