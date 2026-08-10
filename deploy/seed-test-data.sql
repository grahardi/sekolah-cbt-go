-- Contoh data test buat coba alur login -> soal -> jawab -> selesai.
-- Ganti '<paste hash di sini>' dengan hasil dari:
--   ./cbt-server hash-password rahasia123
--
-- Jalankan dengan:
--   psql -U postgres -d golang -f deploy/seed-test-data.sql

SET search_path TO cbt_smpn1turen;

SELECT gen_random_uuid() AS matpel_id \gset
INSERT INTO matpels (id, nama, kode) VALUES (:'matpel_id', 'Matematika', 'MTK');

SELECT gen_random_uuid() AS banksoal_id \gset
INSERT INTO banksoals (id, matpel_id, nama, durasi_menit)
VALUES (:'banksoal_id', :'matpel_id', 'UTS Matematika Kelas 9', 60);

SELECT gen_random_uuid() AS soal_id \gset
INSERT INTO soals (id, banksoal_id, tipe_soal, pertanyaan, urutan)
VALUES (:'soal_id', :'banksoal_id', 1, '2 + 2 = ?', 1);

INSERT INTO jawaban_soals (soal_id, text_jawaban, correct, urutan) VALUES
  (:'soal_id', '3', false, 1),
  (:'soal_id', '4', true, 2),
  (:'soal_id', '5', false, 3);

SELECT gen_random_uuid() AS jadwal_id \gset
INSERT INTO jadwals (id, banksoal_id, nama, mulai_at, selesai_at)
VALUES (:'jadwal_id', :'banksoal_id', 'Sesi 1', now(), now() + interval '2 hours');

INSERT INTO ujians (jadwal_id, status) VALUES (:'jadwal_id', 'aktif');

INSERT INTO tokens (jadwal_id, token, status) VALUES (:'jadwal_id', 'ABC123', 'aktif');

-- ganti password_hash di bawah dengan hasil `./cbt-server hash-password rahasia123`
INSERT INTO pesertas (no_ujian, nama, kelas, password_hash, status)
VALUES ('9999', 'Siswa Test', '9A', '<paste hash di sini>', 1);
