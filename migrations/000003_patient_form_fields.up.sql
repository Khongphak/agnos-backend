-- 000003_patient_form_fields.up.sql
-- Adds 8 optional patient fields and makes national_id nullable to support
-- foreign patients who identify with a passport instead of a Thai national ID.
--
-- Open item: application-layer validation must ensure that every patient row
-- has either national_id OR passport_id populated (constraint cannot be expressed
-- as a simple NOT NULL — requires a CHECK or trigger if enforced in DB).

ALTER TABLE patients
    ADD COLUMN IF NOT EXISTS middle_name_th          TEXT,
    ADD COLUMN IF NOT EXISTS middle_name_en          TEXT,
    ADD COLUMN IF NOT EXISTS passport_id             VARCHAR(20),
    ADD COLUMN IF NOT EXISTS preferred_language      VARCHAR(50),
    ADD COLUMN IF NOT EXISTS nationality             VARCHAR(100),
    ADD COLUMN IF NOT EXISTS emergency_contact_name  TEXT,
    ADD COLUMN IF NOT EXISTS emergency_contact_phone VARCHAR(20),
    ADD COLUMN IF NOT EXISTS religion                VARCHAR(100);

ALTER TABLE patients
    ALTER COLUMN national_id DROP NOT NULL;
