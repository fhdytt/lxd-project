-- Skema Database dari Sistem Manajemen Environment Linux

-- Extension untuk UUID 
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Table modules untuk menyimpan informasi modul
CREATE TABLE modules (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code                VARCHAR(50) UNIQUE NOT NULL,   -- 'netbegin', 'netadmin'
    name                VARCHAR(150) NOT NULL,         -- 'Network Beginner'
    master_container    VARCHAR(100) NOT NULL,         -- 'master-netbegin'
    lxd_profile         VARCHAR(100) NOT NULL,         -- 'praktikum-netbegin'
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Table rooms untuk menyimpan informasi ruangan kursus
CREATE TABLE rooms (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nama         VARCHAR(50) UNIQUE NOT NULL,   -- 'f491', 'f492', 'f4111', 'f4112'
    port_prefix  VARCHAR(10) NOT NULL,          -- '21', '22', '23', '24'
    capacity     INT NOT NULL DEFAULT 5,        -- jumlah container per sesi
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Table praktikan untuk menyimpan informasi praktikan
CREATE TABLE praktikan (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    npm         VARCHAR(30) UNIQUE NOT NULL,
    nama        VARCHAR(150) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Index untuk mengurutkan praktikan berdasarkan NPM
CREATE INDEX idx_praktikan_npm ON praktikan (npm);

-- Table sessions untuk menyimpan informasi sesi praktikum
CREATE TYPE session_status AS ENUM ('scheduled', 'active', 'completed', 'cancelled');

CREATE TABLE sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_code     VARCHAR(50) NOT NULL,                -- id cohort, contoh = '1WADR261014L'
    module_id       UUID NOT NULL REFERENCES modules(id) ON DELETE RESTRICT,
    room_id         UUID NOT NULL REFERENCES rooms(id) ON DELETE RESTRICT,
    meeting_number  INT NOT NULL,                       -- menentukan pertemuan ke berapa 
    session_date    DATE NOT NULL,
    status          session_status NOT NULL DEFAULT 'scheduled',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- satu course_code tidak boleh punya 2 sesi dengan nomor pertemuan yang sama
    UNIQUE (course_code, meeting_number)
);

CREATE INDEX idx_sessions_course_code ON sessions (course_code);
CREATE INDEX idx_sessions_room_date ON sessions (room_id, session_date);
CREATE INDEX idx_sessions_status ON sessions (status);

-- Table environments untuk menyimpan informasi environment container kursus
CREATE TYPE environment_status AS ENUM (
    'provisioning', 'running', 'stopped', 'error', 'reset'
);

CREATE TABLE environments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id      UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    praktikan_id    UUID REFERENCES praktikan(id) ON DELETE SET NULL,  -- nullable, diisi saat identifikasi via TUI

    container_name  VARCHAR(100) UNIQUE NOT NULL,   -- 'f491-01'
    slot_number     INT NOT NULL,                   -- 1, 2, 3, ...
    ssh_port        INT NOT NULL,                   -- 2101, 2102, ...
    status          environment_status NOT NULL DEFAULT 'provisioning',

    has_clean_snapshot  BOOLEAN NOT NULL DEFAULT false,
    identified_at       TIMESTAMPTZ,                  -- waktu praktikan saat submit nama/NPM
    api_token_hash       TEXT,                        -- hash dari token yang dimasukan ke container
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- satu sesi tidak boleh punya slot number yang sama
    UNIQUE (session_id, slot_number),
    -- satu sesi tidak boleh punya port yang sama
    UNIQUE (session_id, ssh_port)
);

CREATE INDEX idx_environments_session ON environments (session_id);
CREATE INDEX idx_environments_praktikan ON environments (praktikan_id);
CREATE INDEX idx_environments_status ON environments (status);

-- Table admins untuk menyimpan informasi admin(Kebutuhan Dashboard Web)
CREATE TABLE admins (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username        VARCHAR(50) UNIQUE NOT NULL,
    password_hash   TEXT NOT NULL,
    full_name       VARCHAR(150),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_login_at   TIMESTAMPTZ
);

-- Trigger untuk mengupdate updated_at saat record diubah dan mencegah perubahan created_at
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_modules_updated_at BEFORE UPDATE ON modules
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_praktikan_updated_at BEFORE UPDATE ON praktikan
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_sessions_updated_at BEFORE UPDATE ON sessions
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_environments_updated_at BEFORE UPDATE ON environments
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
