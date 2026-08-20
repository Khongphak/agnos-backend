INSERT INTO hospitals (code, name) VALUES ('HOSP01', 'Hospital A') ON CONFLICT (code) DO NOTHING;
