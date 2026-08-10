-- Contoh data test buat coba alur login -> soal -> jawab -> selesai.
-- Ganti :password_hash dengan hasil dari:
--   ./cbt-server hash-password rahasia123

SET search_path TO cbt_smpn1turen;

INSERT INTO matpels (id, nama, kode) VALUES
  (gen_random_uuid(), 'Matematika', 'MTK') RETURNING id \gset

INSERT INTO banksoals (id, matpel_id, nama, durasi_menit) VALUES
  (gen_random_uuid(), :'id', 'UTS Matematika Kelas 9', 60) RETURNING id AS banksoal_id \gset

INSERT INTO soals (id, banksoal_id, tipe_soal, pertanyaan, urutan) VALUES
  (gen_random_uuid(), :'banksoal_id', 1, '2 + 2 = ?', 1) RETURNING id AS soal_id \gset

INSERT INTO jawaban_soals (soal_id, text_jawaban, correct, urutan) VALUES
  (:'soal_id', '3', false, 1),
  (:'soal_id', '4', true, 2),
  (:'soal_id', '5', false, 3);

INSERT INTO jadwals (id, banksoal_id, nama, mulai_at, selesai_at) VALUES
  (gen_random_uuid(), :'banksoal_id', 'Sesi 1', now(), now() + interval '2 hours')
  RETURNING id AS jadwal_id \gset

INSERT INTO ujians (jadwal_id, status) VALUES (:'jadwal_id', 'aktif');

INSERT INTO tokens (jadwal_id, token, status) VALUES (:'jadwal_id', 'ABC123', 'aktif');

-- ganti password_hash di bawah dengan hasil `./cbt-server hash-password rahasia123`
INSERT INTO pesertas (no_ujian, nama, kelas, password_hash, status) VALUES
  ('9999', 'Siswa Test', '9A', '<paste hash di sini>', 1);
