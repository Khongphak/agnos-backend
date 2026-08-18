-- 000003_patient_form_fields.down.sql
-- WARNING: restoring NOT NULL on national_id will fail if any row has national_id = NULL.
-- Ensure all rows have national_id populated before rolling back.

ALTER TABLE patients
    ALTER COLUMN national_id SET NOT NULL;

ALTER TABLE patients
    DROP COLUMN IF EXISTS religion,
    DROP COLUMN IF EXISTS emergency_contact_phone,
    DROP COLUMN IF EXISTS emergency_contact_name,
    DROP COLUMN IF EXISTS nationality,
    DROP COLUMN IF EXISTS preferred_language,
    DROP COLUMN IF EXISTS passport_id,
    DROP COLUMN IF EXISTS middle_name_en,
    DROP COLUMN IF EXISTS middle_name_th;
