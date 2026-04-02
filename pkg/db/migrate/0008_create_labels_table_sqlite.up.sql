CREATE TABLE IF NOT EXISTS labels (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    repo_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    color TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (repo_id) REFERENCES repos(id) ON DELETE CASCADE,
    UNIQUE (repo_id, name)
);

CREATE TABLE IF NOT EXISTS issue_labels (
    issue_id INTEGER NOT NULL,
    label_id INTEGER NOT NULL,
    PRIMARY KEY (issue_id, label_id),
    FOREIGN KEY (issue_id) REFERENCES issues(id) ON DELETE CASCADE,
    FOREIGN KEY (label_id) REFERENCES labels(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_labels_repo_id ON labels(repo_id);
