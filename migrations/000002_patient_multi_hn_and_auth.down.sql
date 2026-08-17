-- 000002_patient_multi_hn_and_auth.down.sql
-- WARNING: rolling back loses any patient_hn data stored in patient_hospital_registrations.

-- ── 3. staff_refresh_tokens
DROP TABLE IF EXISTS staff_refresh_tokens;

-- ── 2. staff — auth fields
ALTER TABLE staff
    DROP COLUMN IF EXISTS last_login_at,
    DROP COLUMN IF EXISTS is_active,
    DROP COLUMN IF EXISTS role;

-- ── 1. patient_hospital_registrations → restore flat columns
DROP TABLE IF EXISTS patient_hospital_registrations;

ALTER TABLE patients
    ADD COLUMN IF NOT EXISTS hospital_id BIGINT REFERENCES hospitals(id),
    ADD COLUMN IF NOT EXISTS patient_hn  TEXT;

CREATE INDEX IF NOT EXISTS idx_patients_hospital_id ON patients(hospital_id);
