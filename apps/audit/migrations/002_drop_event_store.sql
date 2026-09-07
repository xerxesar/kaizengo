-- Remove legacy event store tables (models are the source of truth).
DROP TABLE IF EXISTS events;
DROP TABLE IF EXISTS streams;
