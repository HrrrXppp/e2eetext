CREATE TABLE node (
    singleton BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (singleton),
    id        UUID NOT NULL DEFAULT gen_random_uuid()
);

INSERT INTO node DEFAULT VALUES;
