-- Migration 003: Document Metadata
-- Adds a JSON column to store structured metadata from the Document Metadata Extraction Agent

ALTER TABLE user_documents ADD COLUMN metadata_json TEXT DEFAULT '{}';
