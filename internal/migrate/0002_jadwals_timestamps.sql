-- jadwals originally stored tanggal + jam_mulai + jam_selesai separately.
-- Switched to a single timestamptz window: simpler to reason about, and
-- avoids relying on how the Postgres driver maps the bare `time` type.
-- Safe to run as a plain ALTER since the table has no rows yet.

ALTER TABLE jadwals
    DROP COLUMN tanggal,
    DROP COLUMN jam_mulai,
    DROP COLUMN jam_selesai,
    ADD COLUMN mulai_at timestamptz NOT NULL,
    ADD COLUMN selesai_at timestamptz NOT NULL;
