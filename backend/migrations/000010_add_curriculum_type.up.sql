ALTER TABLE institutions
  ADD COLUMN curriculum_type VARCHAR(10) NOT NULL DEFAULT '844'
  CHECK (curriculum_type IN ('844', 'cbc'));
  