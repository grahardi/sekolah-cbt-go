-- Core schema for the ujian (CBT) engine. One of these schemas exists per
-- sekolah, so every table here is created unqualified and resolved through
-- the connection's search_path (see internal/db).
--
-- Table names deliberately echo the legacy Extraordinary CBT
-- (dump-postgres.sql: soals, jawaban_soals, pesertas, tokens, hasil_ujians,
-- ...) so anyone who already knows that data model recognizes this one.
-- Columns are trimmed down to what the Go engine actually uses; per-tipe-soal
-- scoring breakdowns and extras can be added in later migrations once the
-- exam engine needs them.

-- gen_random_uuid() has been built into PostgreSQL core since v13, so no
-- pgcrypto extension is needed here.

CREATE TABLE matpels (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    nama       varchar(255) NOT NULL,
    kode       varchar(50),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE banksoals (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    matpel_id       uuid NOT NULL REFERENCES matpels(id),
    nama            varchar(255) NOT NULL,
    durasi_menit    integer NOT NULL DEFAULT 90,
    acak_soal       boolean NOT NULL DEFAULT true,
    acak_jawaban    boolean NOT NULL DEFAULT true,
    tampilkan_hasil boolean NOT NULL DEFAULT false,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);

-- tipe_soal: 1 Pilihan Ganda | 2 Esai | 3 Listening | 4 Isian Singkat
CREATE TABLE soals (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    banksoal_id uuid NOT NULL REFERENCES banksoals(id) ON DELETE CASCADE,
    tipe_soal   smallint NOT NULL DEFAULT 1,
    pertanyaan  text NOT NULL,
    rujukan     text,
    audio       varchar(255),
    urutan      integer NOT NULL DEFAULT 0,
    poin        numeric(6, 2) NOT NULL DEFAULT 1,
    deleted_at  timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_soals_banksoal_id ON soals (banksoal_id) WHERE deleted_at IS NULL;

CREATE TABLE jawaban_soals (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    soal_id      uuid NOT NULL REFERENCES soals(id) ON DELETE CASCADE,
    text_jawaban text NOT NULL,
    correct      boolean NOT NULL DEFAULT false,
    urutan       integer NOT NULL DEFAULT 0,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_jawaban_soals_soal_id ON jawaban_soals (soal_id);

CREATE TABLE jadwals (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    banksoal_id uuid NOT NULL REFERENCES banksoals(id),
    nama        varchar(255) NOT NULL,
    tanggal     date NOT NULL,
    jam_mulai   time NOT NULL,
    jam_selesai time NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE ujians (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    jadwal_id  uuid NOT NULL REFERENCES jadwals(id),
    status     varchar(20) NOT NULL DEFAULT 'aktif' CHECK (status IN ('aktif', 'selesai')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- Synced from Buku Induk (main sekolah.co.id app), same as the existing
-- Extraordinary CBT student-sync flow.
CREATE TABLE pesertas (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    no_ujian      varchar(50) NOT NULL UNIQUE,
    nama          varchar(255) NOT NULL,
    kelas         varchar(50),
    password_hash varchar(255) NOT NULL,
    status        smallint NOT NULL DEFAULT 1,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE tokens (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    jadwal_id  uuid NOT NULL REFERENCES jadwals(id) ON DELETE CASCADE,
    token      varchar(20) NOT NULL UNIQUE,
    status     varchar(20) NOT NULL DEFAULT 'nonaktif' CHECK (status IN ('aktif', 'nonaktif')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- One row per peserta's attempt at a ujian.
CREATE TABLE siswa_ujians (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    ujian_id      uuid NOT NULL REFERENCES ujians(id),
    peserta_id    uuid NOT NULL REFERENCES pesertas(id),
    waktu_mulai   timestamptz,
    waktu_selesai timestamptz,
    status        varchar(20) NOT NULL DEFAULT 'berlangsung' CHECK (status IN ('berlangsung', 'selesai')),
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (ujian_id, peserta_id)
);

-- Peserta's answer for one soal within one siswa_ujian attempt.
CREATE TABLE jawaban_pesertas (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    siswa_ujian_id  uuid NOT NULL REFERENCES siswa_ujians(id) ON DELETE CASCADE,
    soal_id         uuid NOT NULL REFERENCES soals(id),
    jawaban_soal_id uuid REFERENCES jawaban_soals(id),
    jawaban_text    text,
    is_correct      boolean,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (siswa_ujian_id, soal_id)
);

CREATE TABLE hasil_ujians (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    ujian_id     uuid NOT NULL REFERENCES ujians(id),
    peserta_id   uuid NOT NULL REFERENCES pesertas(id),
    jumlah_benar integer NOT NULL DEFAULT 0,
    jumlah_salah integer NOT NULL DEFAULT 0,
    nilai        numeric(6, 2) NOT NULL DEFAULT 0,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (ujian_id, peserta_id)
);
