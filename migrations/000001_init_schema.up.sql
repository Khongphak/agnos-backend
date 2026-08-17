-- 000001_init_schema.up.sql
-- Assumptions (flag for frontend sync on next API round):
--   * patient_hn is scoped to one hospital per patient record; multi-hospital HN mapping
--     will require a separate junction table if needed later.
--   * gender stores 'male' | 'female' | 'other'; adjust if Hospital A API uses a different
--     encoding (e.g. numeric code).
--   * staff username uniqueness is scoped per hospital (UNIQUE(hospital_id, username)).
--   * hospitals table is introduced to satisfy staff.hospital_id FK; populate it before
--     seeding staff rows.

CREATE TABLE IF NOT EXISTS hospitals (
    id          BIGSERIAL   PRIMARY KEY,
    name        TEXT        NOT NULL,
    code        TEXT        NOT NULL UNIQUE,  -- hospital HIS code / short identifier
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS patients (
    id              BIGSERIAL    PRIMARY KEY,
    national_id     VARCHAR(13)  NOT NULL UNIQUE,  -- Thai 13-digit national ID
    first_name_th   TEXT         NOT NULL,
    last_name_th    TEXT         NOT NULL,
    first_name_en   TEXT,
    last_name_en    TEXT,
    date_of_birth   DATE         NOT NULL,
    gender          VARCHAR(10)  NOT NULL CHECK (gender IN ('male', 'female', 'other')),
    phone_number    VARCHAR(20),
    email           VARCHAR(255),
    address         TEXT,
    patient_hn      TEXT,                          -- hospital-assigned record number (assumption: single hospital)
    hospital_id     BIGINT       REFERENCES hospitals(id),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- national_id UNIQUE constraint already creates an implicit index.
CREATE INDEX IF NOT EXISTS idx_patients_hospital_id ON patients(hospital_id);

CREATE TABLE IF NOT EXISTS staff (
    id              BIGSERIAL   PRIMARY KEY,
    hospital_id     BIGINT      NOT NULL REFERENCES hospitals(id),
    username        TEXT        NOT NULL,
    password_hash   TEXT        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (hospital_id, username)              -- username unique within a hospital
);

CREATE INDEX IF NOT EXISTS idx_staff_hospital_id ON staff(hospital_id);
