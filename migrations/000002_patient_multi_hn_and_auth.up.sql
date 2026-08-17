-- 000002_patient_multi_hn_and_auth.up.sql
-- Resolves confirmed assumptions from 000001:
--   1. patient_hn: 1 patient → N hospitals (junction table replaces flat column)
--   2. staff auth: role, is_active, last_login_at + refresh token table

-- ── 1. patient_hospital_registrations ────────────────────────────────────────
--    Replaces the flat patient_hn / hospital_id columns on patients.
--    One row per (patient, hospital) pair; patient_hn is unique within a hospital.

CREATE TABLE IF NOT EXISTS patient_hospital_registrations (
    id          BIGSERIAL   PRIMARY KEY,
    patient_id  BIGINT      NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    hospital_id BIGINT      NOT NULL REFERENCES hospitals(id),
    patient_hn  TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (hospital_id, patient_hn),   -- HN unique within a hospital
    UNIQUE (patient_id, hospital_id)    -- one registration per patient per hospital
);

CREATE INDEX IF NOT EXISTS idx_phr_patient_id  ON patient_hospital_registrations(patient_id);
CREATE INDEX IF NOT EXISTS idx_phr_hospital_id ON patient_hospital_registrations(hospital_id);

-- Remove flat columns now replaced by the junction table
DROP INDEX  IF EXISTS idx_patients_hospital_id;
ALTER TABLE patients DROP COLUMN IF EXISTS patient_hn;
ALTER TABLE patients DROP COLUMN IF EXISTS hospital_id;

-- ── 2. staff — auth fields ────────────────────────────────────────────────────

ALTER TABLE staff
    ADD COLUMN IF NOT EXISTS role          VARCHAR(20)  NOT NULL DEFAULT 'staff'
        CHECK (role IN ('admin', 'staff')),
    ADD COLUMN IF NOT EXISTS is_active     BOOLEAN      NOT NULL DEFAULT true,
    ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ;

-- ── 3. staff_refresh_tokens ───────────────────────────────────────────────────
--    Stores hashed JWT refresh tokens for session management.
--    token_hash: bcrypt/sha256 of the opaque token sent to the client.

CREATE TABLE IF NOT EXISTS staff_refresh_tokens (
    id          BIGSERIAL   PRIMARY KEY,
    staff_id    BIGINT      NOT NULL REFERENCES staff(id) ON DELETE CASCADE,
    token_hash  TEXT        NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_srt_staff_id ON staff_refresh_tokens(staff_id);
