CREATE UNIQUE INDEX IF NOT EXISTS idx_teams_project_slug
    ON teams (project_id, slug);
