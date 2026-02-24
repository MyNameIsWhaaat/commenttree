-- 0001_init.up.sql

CREATE TABLE comments (
  id         BIGSERIAL PRIMARY KEY,
  parent_id  BIGINT NOT NULL DEFAULT 0,
  text       TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  search_tsv tsvector GENERATED ALWAYS AS (to_tsvector('simple', coalesce(text, ''))) STORED,
  CONSTRAINT comments_no_self_parent CHECK (parent_id <> id)
);

CREATE INDEX idx_comments_parent_id ON comments(parent_id);
CREATE INDEX idx_comments_search_tsv ON comments USING GIN (search_tsv);