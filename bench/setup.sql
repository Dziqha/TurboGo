-- TurboGo benchmark DB schema (Techempower-style), idempotent
CREATE USER benchmark WITH PASSWORD 'benchmark';
CREATE DATABASE hello_world OWNER benchmark;
\c hello_world
DROP TABLE IF EXISTS fortune;
DROP TABLE IF EXISTS world;
CREATE TABLE world (
    id integer PRIMARY KEY,
    randomnumber integer NOT NULL
);
INSERT INTO world (id, randomnumber)
SELECT i, (random() * 10000)::int + 1 FROM generate_series(1, 10000) AS i;
CREATE TABLE fortune (
    id integer PRIMARY KEY,
    message text NOT NULL
);
INSERT INTO fortune (id, message) VALUES
(1,  'fortune: No snowflake in an avalanche ever feels responsible.'),
(2,  'fortune: A computer once beat me at chess, but it was no match for me at kick boxing.'),
(3,  'fortune: A good way to keep healthy is to eat more Chinese food.'),
(4,  'fortune: A bad penny always turns up.'),
(5,  'fortune: A bargain is something you don''t need at a price you can''t resist.'),
(6,  'fortune: A bird in the hand is worth two in the bush.'),
(7,  'fortune: A bus is a vehicle that runs twice as fast when you are after it.'),
(8,  'fortune: A car is not for the faint-hearted.'),
(9,  'fortune: A day without sunshine is like, you know, night.'),
(10, 'fortune: A dog is a man''s best friend.'),
(11, 'fortune: A fine wine improves with age, but I don''t.'),
(12, 'fortune: A fool and his money are soon parted.');
ALTER TABLE world OWNER TO benchmark;
ALTER TABLE fortune OWNER TO benchmark;