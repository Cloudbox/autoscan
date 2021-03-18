CREATE TABLE IF NOT EXISTS scan (
    "folder" TEXT NOT NULL,
    "priority" INTEGER NOT NULL,
    "time" DATETIME NOT NULL,
    PRIMARY KEY(folder)
)