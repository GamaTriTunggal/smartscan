-- +goose Up

-- ============================================
-- SEED DATA: LOCATIONS
-- Countries, Provinces, Cities (ASEAN focus)
-- ============================================

-- ============================================

-- ============================================
-- 1. COUNTRIES (ASEAN)
-- ============================================

INSERT INTO countries (code, name, phone_code) VALUES
('ID', 'Indonesia', '+62'),
('MY', 'Malaysia', '+60'),
('PH', 'Philippines', '+63'),
('SG', 'Singapore', '+65'),
('TH', 'Thailand', '+66'),
('VN', 'Vietnam', '+84')
ON CONFLICT (code) DO NOTHING;

-- ============================================
-- 2. PROVINCES & CITIES (COMPREHENSIVE)
-- ============================================

-- ============================================
-- INDONESIA - 38 PROVINCES
-- ============================================

INSERT INTO provinces (country_code, name, code) VALUES
-- Sumatera
('ID', 'Aceh', 'AC'),
('ID', 'Sumatera Utara', 'SU'),
('ID', 'Sumatera Barat', 'SB'),
('ID', 'Riau', 'RI'),
('ID', 'Kepulauan Riau', 'KR'),
('ID', 'Jambi', 'JA'),
('ID', 'Sumatera Selatan', 'SS'),
('ID', 'Kepulauan Bangka Belitung', 'BB'),
('ID', 'Bengkulu', 'BE'),
('ID', 'Lampung', 'LA'),
-- Jawa
('ID', 'DKI Jakarta', 'JK'),
('ID', 'Jawa Barat', 'JB'),
('ID', 'Banten', 'BT'),
('ID', 'Jawa Tengah', 'JT'),
('ID', 'DI Yogyakarta', 'YO'),
('ID', 'Jawa Timur', 'JI'),
-- Kalimantan
('ID', 'Kalimantan Barat', 'KB'),
('ID', 'Kalimantan Tengah', 'KT'),
('ID', 'Kalimantan Selatan', 'KS'),
('ID', 'Kalimantan Timur', 'KI'),
('ID', 'Kalimantan Utara', 'KU'),
-- Sulawesi
('ID', 'Sulawesi Utara', 'SA'),
('ID', 'Gorontalo', 'GO'),
('ID', 'Sulawesi Tengah', 'ST'),
('ID', 'Sulawesi Barat', 'SR'),
('ID', 'Sulawesi Selatan', 'SN'),
('ID', 'Sulawesi Tenggara', 'SG'),
-- Bali & Nusa Tenggara
('ID', 'Bali', 'BA'),
('ID', 'Nusa Tenggara Barat', 'NB'),
('ID', 'Nusa Tenggara Timur', 'NT'),
-- Maluku
('ID', 'Maluku', 'MA'),
('ID', 'Maluku Utara', 'MU'),
-- Papua
('ID', 'Papua', 'PA'),
('ID', 'Papua Barat', 'PB'),
('ID', 'Papua Selatan', 'PS'),
('ID', 'Papua Tengah', 'PT'),
('ID', 'Papua Pegunungan', 'PP'),
('ID', 'Papua Barat Daya', 'PD')
ON CONFLICT DO NOTHING;

-- ============================================
-- INDONESIA CITIES - BY PROVINCE
-- ============================================

-- ACEH
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'ID', city_name, postal FROM provinces p,
(VALUES
  ('Kota Banda Aceh', '23'), ('Kota Sabang', '23'), ('Kota Langsa', '24'), ('Kota Lhokseumawe', '24'),
  ('Kota Subulussalam', '24'), ('Kabupaten Aceh Besar', '23'), ('Kabupaten Aceh Barat', '23'),
  ('Kabupaten Aceh Selatan', '23'), ('Kabupaten Aceh Singkil', '24'), ('Kabupaten Aceh Tengah', '24'),
  ('Kabupaten Aceh Tenggara', '24'), ('Kabupaten Aceh Timur', '24'), ('Kabupaten Aceh Utara', '24'),
  ('Kabupaten Pidie', '24'), ('Kabupaten Bireuen', '24'), ('Kabupaten Simeulue', '23')
) AS cities(city_name, postal)
WHERE p.country_code = 'ID' AND p.name = 'Aceh'
ON CONFLICT DO NOTHING;

-- SUMATERA UTARA
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'ID', city_name, postal FROM provinces p,
(VALUES
  ('Kota Medan', '20'), ('Kota Binjai', '20'), ('Kota Pematang Siantar', '21'), ('Kota Tebing Tinggi', '20'),
  ('Kota Tanjung Balai', '21'), ('Kota Sibolga', '22'), ('Kota Padang Sidempuan', '22'), ('Kota Gunung Sitoli', '22'),
  ('Kabupaten Deli Serdang', '20'), ('Kabupaten Langkat', '20'), ('Kabupaten Karo', '22'), ('Kabupaten Simalungun', '21'),
  ('Kabupaten Asahan', '21'), ('Kabupaten Labuhan Batu', '21'), ('Kabupaten Tapanuli Utara', '22'),
  ('Kabupaten Tapanuli Tengah', '22'), ('Kabupaten Tapanuli Selatan', '22'), ('Kabupaten Nias', '22'),
  ('Kabupaten Mandailing Natal', '22'), ('Kabupaten Toba Samosir', '22'), ('Kabupaten Dairi', '22'),
  ('Kabupaten Samosir', '22'), ('Kabupaten Serdang Bedagai', '20'), ('Kabupaten Batu Bara', '21')
) AS cities(city_name, postal)
WHERE p.country_code = 'ID' AND p.name = 'Sumatera Utara'
ON CONFLICT DO NOTHING;

-- SUMATERA BARAT
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'ID', city_name, postal FROM provinces p,
(VALUES
  ('Kota Padang', '25'), ('Kota Bukittinggi', '26'), ('Kota Padang Panjang', '27'), ('Kota Solok', '27'),
  ('Kota Sawahlunto', '27'), ('Kota Payakumbuh', '26'), ('Kota Pariaman', '25'), ('Kabupaten Pesisir Selatan', '25'),
  ('Kabupaten Solok', '27'), ('Kabupaten Sijunjung', '27'), ('Kabupaten Tanah Datar', '27'),
  ('Kabupaten Padang Pariaman', '25'), ('Kabupaten Agam', '26'), ('Kabupaten Lima Puluh Kota', '26'),
  ('Kabupaten Pasaman', '26'), ('Kabupaten Pasaman Barat', '26'), ('Kabupaten Dharmasraya', '27'),
  ('Kabupaten Solok Selatan', '27'), ('Kabupaten Kepulauan Mentawai', '25')
) AS cities(city_name, postal)
WHERE p.country_code = 'ID' AND p.name = 'Sumatera Barat'
ON CONFLICT DO NOTHING;

-- RIAU
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'ID', city_name, postal FROM provinces p,
(VALUES
  ('Kota Pekanbaru', '28'), ('Kota Dumai', '28'), ('Kabupaten Kampar', '28'), ('Kabupaten Indragiri Hulu', '29'),
  ('Kabupaten Indragiri Hilir', '29'), ('Kabupaten Bengkalis', '28'), ('Kabupaten Rokan Hulu', '28'),
  ('Kabupaten Rokan Hilir', '28'), ('Kabupaten Siak', '28'), ('Kabupaten Kuantan Singingi', '29'),
  ('Kabupaten Pelalawan', '28'), ('Kabupaten Kepulauan Meranti', '28')
) AS cities(city_name, postal)
WHERE p.country_code = 'ID' AND p.name = 'Riau'
ON CONFLICT DO NOTHING;

-- KEPULAUAN RIAU
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'ID', city_name, postal FROM provinces p,
(VALUES
  ('Kota Batam', '29'), ('Kota Tanjung Pinang', '29'), ('Kabupaten Bintan', '29'), ('Kabupaten Karimun', '29'),
  ('Kabupaten Natuna', '29'), ('Kabupaten Lingga', '29'), ('Kabupaten Kepulauan Anambas', '29')
) AS cities(city_name, postal)
WHERE p.country_code = 'ID' AND p.name = 'Kepulauan Riau'
ON CONFLICT DO NOTHING;

-- JAMBI
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'ID', city_name, postal FROM provinces p,
(VALUES
  ('Kota Jambi', '36'), ('Kota Sungai Penuh', '37'), ('Kabupaten Kerinci', '37'), ('Kabupaten Merangin', '37'),
  ('Kabupaten Sarolangun', '37'), ('Kabupaten Batang Hari', '36'), ('Kabupaten Muaro Jambi', '36'),
  ('Kabupaten Tanjung Jabung Timur', '36'), ('Kabupaten Tanjung Jabung Barat', '36'),
  ('Kabupaten Tebo', '37'), ('Kabupaten Bungo', '37')
) AS cities(city_name, postal)
WHERE p.country_code = 'ID' AND p.name = 'Jambi'
ON CONFLICT DO NOTHING;

-- SUMATERA SELATAN
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'ID', city_name, postal FROM provinces p,
(VALUES
  ('Kota Palembang', '30'), ('Kota Prabumulih', '31'), ('Kota Pagar Alam', '31'), ('Kota Lubuk Linggau', '31'),
  ('Kabupaten Ogan Komering Ulu', '32'), ('Kabupaten Ogan Komering Ilir', '30'), ('Kabupaten Muara Enim', '31'),
  ('Kabupaten Lahat', '31'), ('Kabupaten Musi Rawas', '31'), ('Kabupaten Musi Banyuasin', '30'),
  ('Kabupaten Banyuasin', '30'), ('Kabupaten OKU Selatan', '32'), ('Kabupaten OKU Timur', '32'),
  ('Kabupaten Ogan Ilir', '30'), ('Kabupaten Empat Lawang', '31'), ('Kabupaten PALI', '31'),
  ('Kabupaten Musi Rawas Utara', '31')
) AS cities(city_name, postal)
WHERE p.country_code = 'ID' AND p.name = 'Sumatera Selatan'
ON CONFLICT DO NOTHING;

-- KEPULAUAN BANGKA BELITUNG
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'ID', city_name, postal FROM provinces p,
(VALUES
  ('Kota Pangkal Pinang', '33'), ('Kabupaten Bangka', '33'), ('Kabupaten Bangka Barat', '33'),
  ('Kabupaten Bangka Tengah', '33'), ('Kabupaten Bangka Selatan', '33'), ('Kabupaten Belitung', '33'),
  ('Kabupaten Belitung Timur', '33')
) AS cities(city_name, postal)
WHERE p.country_code = 'ID' AND p.name = 'Kepulauan Bangka Belitung'
ON CONFLICT DO NOTHING;

-- BENGKULU
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'ID', city_name, postal FROM provinces p,
(VALUES
  ('Kota Bengkulu', '38'), ('Kabupaten Bengkulu Selatan', '38'), ('Kabupaten Bengkulu Utara', '38'),
  ('Kabupaten Rejang Lebong', '39'), ('Kabupaten Lebong', '39'), ('Kabupaten Kepahiang', '39'),
  ('Kabupaten Mukomuko', '38'), ('Kabupaten Seluma', '38'), ('Kabupaten Kaur', '38'), ('Kabupaten Bengkulu Tengah', '38')
) AS cities(city_name, postal)
WHERE p.country_code = 'ID' AND p.name = 'Bengkulu'
ON CONFLICT DO NOTHING;

-- LAMPUNG
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'ID', city_name, postal FROM provinces p,
(VALUES
  ('Kota Bandar Lampung', '35'), ('Kota Metro', '34'), ('Kabupaten Lampung Selatan', '35'),
  ('Kabupaten Lampung Tengah', '34'), ('Kabupaten Lampung Utara', '34'), ('Kabupaten Lampung Barat', '34'),
  ('Kabupaten Lampung Timur', '34'), ('Kabupaten Tanggamus', '35'), ('Kabupaten Tulang Bawang', '34'),
  ('Kabupaten Way Kanan', '34'), ('Kabupaten Pesawaran', '35'), ('Kabupaten Pringsewu', '35'),
  ('Kabupaten Mesuji', '34'), ('Kabupaten Tulang Bawang Barat', '34'), ('Kabupaten Pesisir Barat', '35')
) AS cities(city_name, postal)
WHERE p.country_code = 'ID' AND p.name = 'Lampung'
ON CONFLICT DO NOTHING;

-- DKI JAKARTA
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'ID', city_name, postal FROM provinces p,
(VALUES
  ('Kota Jakarta Pusat', '10'), ('Kota Jakarta Utara', '14'), ('Kota Jakarta Barat', '11'),
  ('Kota Jakarta Selatan', '12'), ('Kota Jakarta Timur', '13'), ('Kabupaten Kepulauan Seribu', '14')
) AS cities(city_name, postal)
WHERE p.country_code = 'ID' AND p.name = 'DKI Jakarta'
ON CONFLICT DO NOTHING;

-- JAWA BARAT
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'ID', city_name, postal FROM provinces p,
(VALUES
  ('Kota Bandung', '40'), ('Kota Bekasi', '17'), ('Kota Bogor', '16'), ('Kota Cirebon', '45'),
  ('Kota Depok', '16'), ('Kota Sukabumi', '43'), ('Kota Cimahi', '40'), ('Kota Tasikmalaya', '46'),
  ('Kota Banjar', '46'), ('Kabupaten Bandung', '40'), ('Kabupaten Bandung Barat', '40'),
  ('Kabupaten Bekasi', '17'), ('Kabupaten Bogor', '16'), ('Kabupaten Ciamis', '46'),
  ('Kabupaten Cianjur', '43'), ('Kabupaten Cirebon', '45'), ('Kabupaten Garut', '44'),
  ('Kabupaten Indramayu', '45'), ('Kabupaten Karawang', '41'), ('Kabupaten Kuningan', '45'),
  ('Kabupaten Majalengka', '45'), ('Kabupaten Pangandaran', '46'), ('Kabupaten Purwakarta', '41'),
  ('Kabupaten Subang', '41'), ('Kabupaten Sukabumi', '43'), ('Kabupaten Sumedang', '45'),
  ('Kabupaten Tasikmalaya', '46')
) AS cities(city_name, postal)
WHERE p.country_code = 'ID' AND p.name = 'Jawa Barat'
ON CONFLICT DO NOTHING;

-- BANTEN
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'ID', city_name, postal FROM provinces p,
(VALUES
  ('Kota Tangerang', '15'), ('Kota Tangerang Selatan', '15'), ('Kota Serang', '42'), ('Kota Cilegon', '42'),
  ('Kabupaten Tangerang', '15'), ('Kabupaten Serang', '42'), ('Kabupaten Pandeglang', '42'), ('Kabupaten Lebak', '42')
) AS cities(city_name, postal)
WHERE p.country_code = 'ID' AND p.name = 'Banten'
ON CONFLICT DO NOTHING;

-- JAWA TENGAH
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'ID', city_name, postal FROM provinces p,
(VALUES
  ('Kota Semarang', '50'), ('Kota Surakarta', '57'), ('Kota Magelang', '56'), ('Kota Salatiga', '50'),
  ('Kota Pekalongan', '51'), ('Kota Tegal', '52'), ('Kabupaten Semarang', '50'), ('Kabupaten Kendal', '51'),
  ('Kabupaten Demak', '59'), ('Kabupaten Grobogan', '58'), ('Kabupaten Pekalongan', '51'),
  ('Kabupaten Batang', '51'), ('Kabupaten Tegal', '52'), ('Kabupaten Brebes', '52'),
  ('Kabupaten Pemalang', '52'), ('Kabupaten Purbalingga', '53'), ('Kabupaten Banjarnegara', '53'),
  ('Kabupaten Banyumas', '53'), ('Kabupaten Cilacap', '53'), ('Kabupaten Kebumen', '54'),
  ('Kabupaten Purworejo', '54'), ('Kabupaten Wonosobo', '56'), ('Kabupaten Magelang', '56'),
  ('Kabupaten Temanggung', '56'), ('Kabupaten Boyolali', '57'), ('Kabupaten Klaten', '57'),
  ('Kabupaten Sukoharjo', '57'), ('Kabupaten Karanganyar', '57'), ('Kabupaten Wonogiri', '57'),
  ('Kabupaten Sragen', '57'), ('Kabupaten Blora', '58'), ('Kabupaten Rembang', '59'),
  ('Kabupaten Pati', '59'), ('Kabupaten Kudus', '59'), ('Kabupaten Jepara', '59')
) AS cities(city_name, postal)
WHERE p.country_code = 'ID' AND p.name = 'Jawa Tengah'
ON CONFLICT DO NOTHING;

-- DI YOGYAKARTA
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'ID', city_name, postal FROM provinces p,
(VALUES
  ('Kota Yogyakarta', '55'), ('Kabupaten Sleman', '55'), ('Kabupaten Bantul', '55'),
  ('Kabupaten Kulon Progo', '55'), ('Kabupaten Gunung Kidul', '55')
) AS cities(city_name, postal)
WHERE p.country_code = 'ID' AND p.name = 'DI Yogyakarta'
ON CONFLICT DO NOTHING;

-- JAWA TIMUR
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'ID', city_name, postal FROM provinces p,
(VALUES
  ('Kota Surabaya', '60'), ('Kota Malang', '65'), ('Kota Kediri', '64'), ('Kota Blitar', '66'),
  ('Kota Mojokerto', '61'), ('Kota Madiun', '63'), ('Kota Pasuruan', '67'), ('Kota Probolinggo', '67'),
  ('Kota Batu', '65'), ('Kabupaten Sidoarjo', '61'), ('Kabupaten Gresik', '61'),
  ('Kabupaten Lamongan', '62'), ('Kabupaten Tuban', '62'), ('Kabupaten Bojonegoro', '62'),
  ('Kabupaten Ngawi', '63'), ('Kabupaten Magetan', '63'), ('Kabupaten Madiun', '63'),
  ('Kabupaten Ponorogo', '63'), ('Kabupaten Pacitan', '63'), ('Kabupaten Trenggalek', '66'),
  ('Kabupaten Tulungagung', '66'), ('Kabupaten Blitar', '66'), ('Kabupaten Kediri', '64'),
  ('Kabupaten Nganjuk', '64'), ('Kabupaten Jombang', '61'), ('Kabupaten Mojokerto', '61'),
  ('Kabupaten Malang', '65'), ('Kabupaten Pasuruan', '67'), ('Kabupaten Probolinggo', '67'),
  ('Kabupaten Lumajang', '67'), ('Kabupaten Jember', '68'), ('Kabupaten Bondowoso', '68'),
  ('Kabupaten Situbondo', '68'), ('Kabupaten Banyuwangi', '68'), ('Kabupaten Bangkalan', '69'),
  ('Kabupaten Sampang', '69'), ('Kabupaten Pamekasan', '69'), ('Kabupaten Sumenep', '69')
) AS cities(city_name, postal)
WHERE p.country_code = 'ID' AND p.name = 'Jawa Timur'
ON CONFLICT DO NOTHING;

-- KALIMANTAN BARAT
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'ID', city_name, postal FROM provinces p,
(VALUES
  ('Kota Pontianak', '78'), ('Kota Singkawang', '79'), ('Kabupaten Sambas', '79'), ('Kabupaten Bengkayang', '79'),
  ('Kabupaten Landak', '78'), ('Kabupaten Mempawah', '78'), ('Kabupaten Sanggau', '78'), ('Kabupaten Ketapang', '78'),
  ('Kabupaten Sintang', '78'), ('Kabupaten Kapuas Hulu', '78'), ('Kabupaten Sekadau', '79'),
  ('Kabupaten Melawi', '78'), ('Kabupaten Kayong Utara', '78'), ('Kabupaten Kubu Raya', '78')
) AS cities(city_name, postal)
WHERE p.country_code = 'ID' AND p.name = 'Kalimantan Barat'
ON CONFLICT DO NOTHING;

-- KALIMANTAN TENGAH
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'ID', city_name, postal FROM provinces p,
(VALUES
  ('Kota Palangka Raya', '73'), ('Kabupaten Kotawaringin Barat', '74'), ('Kabupaten Kotawaringin Timur', '74'),
  ('Kabupaten Kapuas', '73'), ('Kabupaten Barito Selatan', '73'), ('Kabupaten Barito Utara', '73'),
  ('Kabupaten Sukamara', '74'), ('Kabupaten Lamandau', '74'), ('Kabupaten Seruyan', '74'),
  ('Kabupaten Katingan', '74'), ('Kabupaten Pulang Pisau', '74'), ('Kabupaten Gunung Mas', '74'),
  ('Kabupaten Barito Timur', '73'), ('Kabupaten Murung Raya', '73')
) AS cities(city_name, postal)
WHERE p.country_code = 'ID' AND p.name = 'Kalimantan Tengah'
ON CONFLICT DO NOTHING;

-- KALIMANTAN SELATAN
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'ID', city_name, postal FROM provinces p,
(VALUES
  ('Kota Banjarmasin', '70'), ('Kota Banjarbaru', '70'), ('Kabupaten Banjar', '70'), ('Kabupaten Barito Kuala', '70'),
  ('Kabupaten Tapin', '71'), ('Kabupaten Hulu Sungai Selatan', '71'), ('Kabupaten Hulu Sungai Tengah', '71'),
  ('Kabupaten Hulu Sungai Utara', '71'), ('Kabupaten Balangan', '71'), ('Kabupaten Tabalong', '71'),
  ('Kabupaten Tanah Laut', '70'), ('Kabupaten Tanah Bumbu', '72'), ('Kabupaten Kotabaru', '72')
) AS cities(city_name, postal)
WHERE p.country_code = 'ID' AND p.name = 'Kalimantan Selatan'
ON CONFLICT DO NOTHING;

-- KALIMANTAN TIMUR
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'ID', city_name, postal FROM provinces p,
(VALUES
  ('Kota Samarinda', '75'), ('Kota Balikpapan', '76'), ('Kota Bontang', '75'), ('Kabupaten Kutai Kartanegara', '75'),
  ('Kabupaten Kutai Barat', '75'), ('Kabupaten Kutai Timur', '75'), ('Kabupaten Berau', '77'),
  ('Kabupaten Paser', '76'), ('Kabupaten Penajam Paser Utara', '76'), ('Kabupaten Mahakam Ulu', '75')
) AS cities(city_name, postal)
WHERE p.country_code = 'ID' AND p.name = 'Kalimantan Timur'
ON CONFLICT DO NOTHING;

-- KALIMANTAN UTARA
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'ID', city_name, postal FROM provinces p,
(VALUES
  ('Kota Tarakan', '77'), ('Kabupaten Bulungan', '77'), ('Kabupaten Malinau', '77'),
  ('Kabupaten Nunukan', '77'), ('Kabupaten Tana Tidung', '77')
) AS cities(city_name, postal)
WHERE p.country_code = 'ID' AND p.name = 'Kalimantan Utara'
ON CONFLICT DO NOTHING;

-- SULAWESI UTARA
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'ID', city_name, postal FROM provinces p,
(VALUES
  ('Kota Manado', '95'), ('Kota Bitung', '95'), ('Kota Tomohon', '95'), ('Kota Kotamobagu', '95'),
  ('Kabupaten Minahasa', '95'), ('Kabupaten Minahasa Utara', '95'), ('Kabupaten Minahasa Selatan', '95'),
  ('Kabupaten Minahasa Tenggara', '95'), ('Kabupaten Bolaang Mongondow', '95'),
  ('Kabupaten Bolaang Mongondow Utara', '95'), ('Kabupaten Bolaang Mongondow Selatan', '95'),
  ('Kabupaten Bolaang Mongondow Timur', '95'), ('Kabupaten Kepulauan Sangihe', '95'),
  ('Kabupaten Kepulauan Talaud', '95'), ('Kabupaten Kepulauan Siau Tagulandang Biaro', '95')
) AS cities(city_name, postal)
WHERE p.country_code = 'ID' AND p.name = 'Sulawesi Utara'
ON CONFLICT DO NOTHING;

-- GORONTALO
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'ID', city_name, postal FROM provinces p,
(VALUES
  ('Kota Gorontalo', '96'), ('Kabupaten Gorontalo', '96'), ('Kabupaten Gorontalo Utara', '96'),
  ('Kabupaten Boalemo', '96'), ('Kabupaten Bone Bolango', '96'), ('Kabupaten Pohuwato', '96')
) AS cities(city_name, postal)
WHERE p.country_code = 'ID' AND p.name = 'Gorontalo'
ON CONFLICT DO NOTHING;

-- SULAWESI TENGAH
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'ID', city_name, postal FROM provinces p,
(VALUES
  ('Kota Palu', '94'), ('Kabupaten Donggala', '94'), ('Kabupaten Parigi Moutong', '94'), ('Kabupaten Sigi', '94'),
  ('Kabupaten Poso', '94'), ('Kabupaten Tojo Una-Una', '94'), ('Kabupaten Toli-Toli', '94'),
  ('Kabupaten Buol', '94'), ('Kabupaten Banggai', '94'), ('Kabupaten Banggai Kepulauan', '94'),
  ('Kabupaten Banggai Laut', '94'), ('Kabupaten Morowali', '94'), ('Kabupaten Morowali Utara', '94')
) AS cities(city_name, postal)
WHERE p.country_code = 'ID' AND p.name = 'Sulawesi Tengah'
ON CONFLICT DO NOTHING;

-- SULAWESI BARAT
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'ID', city_name, postal FROM provinces p,
(VALUES
  ('Kabupaten Majene', '91'), ('Kabupaten Polewali Mandar', '91'), ('Kabupaten Mamasa', '91'),
  ('Kabupaten Mamuju', '91'), ('Kabupaten Mamuju Utara', '91'), ('Kabupaten Mamuju Tengah', '91')
) AS cities(city_name, postal)
WHERE p.country_code = 'ID' AND p.name = 'Sulawesi Barat'
ON CONFLICT DO NOTHING;

-- SULAWESI SELATAN
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'ID', city_name, postal FROM provinces p,
(VALUES
  ('Kota Makassar', '90'), ('Kota Parepare', '91'), ('Kota Palopo', '91'), ('Kabupaten Gowa', '92'),
  ('Kabupaten Takalar', '92'), ('Kabupaten Jeneponto', '92'), ('Kabupaten Bantaeng', '92'),
  ('Kabupaten Bulukumba', '92'), ('Kabupaten Selayar', '92'), ('Kabupaten Sinjai', '92'),
  ('Kabupaten Bone', '92'), ('Kabupaten Maros', '90'), ('Kabupaten Pangkep', '90'),
  ('Kabupaten Barru', '90'), ('Kabupaten Soppeng', '90'), ('Kabupaten Wajo', '90'),
  ('Kabupaten Sidenreng Rappang', '91'), ('Kabupaten Pinrang', '91'), ('Kabupaten Enrekang', '91'),
  ('Kabupaten Luwu', '91'), ('Kabupaten Luwu Utara', '92'), ('Kabupaten Luwu Timur', '92'),
  ('Kabupaten Tana Toraja', '91'), ('Kabupaten Toraja Utara', '91')
) AS cities(city_name, postal)
WHERE p.country_code = 'ID' AND p.name = 'Sulawesi Selatan'
ON CONFLICT DO NOTHING;

-- SULAWESI TENGGARA
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'ID', city_name, postal FROM provinces p,
(VALUES
  ('Kota Kendari', '93'), ('Kota Bau-Bau', '93'), ('Kabupaten Konawe', '93'), ('Kabupaten Konawe Selatan', '93'),
  ('Kabupaten Konawe Utara', '93'), ('Kabupaten Konawe Kepulauan', '93'), ('Kabupaten Kolaka', '93'),
  ('Kabupaten Kolaka Utara', '93'), ('Kabupaten Kolaka Timur', '93'), ('Kabupaten Bombana', '93'),
  ('Kabupaten Wakatobi', '93'), ('Kabupaten Muna', '93'), ('Kabupaten Muna Barat', '93'),
  ('Kabupaten Buton', '93'), ('Kabupaten Buton Utara', '93'), ('Kabupaten Buton Selatan', '93'),
  ('Kabupaten Buton Tengah', '93')
) AS cities(city_name, postal)
WHERE p.country_code = 'ID' AND p.name = 'Sulawesi Tenggara'
ON CONFLICT DO NOTHING;

-- BALI
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'ID', city_name, postal FROM provinces p,
(VALUES
  ('Kota Denpasar', '80'), ('Kabupaten Badung', '80'), ('Kabupaten Gianyar', '80'), ('Kabupaten Tabanan', '82'),
  ('Kabupaten Klungkung', '80'), ('Kabupaten Bangli', '80'), ('Kabupaten Karangasem', '80'),
  ('Kabupaten Buleleng', '81'), ('Kabupaten Jembrana', '82')
) AS cities(city_name, postal)
WHERE p.country_code = 'ID' AND p.name = 'Bali'
ON CONFLICT DO NOTHING;

-- NUSA TENGGARA BARAT
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'ID', city_name, postal FROM provinces p,
(VALUES
  ('Kota Mataram', '83'), ('Kota Bima', '84'), ('Kabupaten Lombok Barat', '83'), ('Kabupaten Lombok Tengah', '83'),
  ('Kabupaten Lombok Timur', '83'), ('Kabupaten Lombok Utara', '83'), ('Kabupaten Sumbawa', '84'),
  ('Kabupaten Sumbawa Barat', '84'), ('Kabupaten Dompu', '84'), ('Kabupaten Bima', '84')
) AS cities(city_name, postal)
WHERE p.country_code = 'ID' AND p.name = 'Nusa Tenggara Barat'
ON CONFLICT DO NOTHING;

-- NUSA TENGGARA TIMUR
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'ID', city_name, postal FROM provinces p,
(VALUES
  ('Kota Kupang', '85'), ('Kabupaten Kupang', '85'), ('Kabupaten Timor Tengah Selatan', '85'),
  ('Kabupaten Timor Tengah Utara', '85'), ('Kabupaten Belu', '85'), ('Kabupaten Malaka', '85'),
  ('Kabupaten Alor', '85'), ('Kabupaten Lembata', '86'), ('Kabupaten Flores Timur', '86'),
  ('Kabupaten Sikka', '86'), ('Kabupaten Ende', '86'), ('Kabupaten Nagekeo', '86'),
  ('Kabupaten Ngada', '86'), ('Kabupaten Manggarai', '86'), ('Kabupaten Manggarai Barat', '86'),
  ('Kabupaten Manggarai Timur', '86'), ('Kabupaten Sumba Timur', '87'), ('Kabupaten Sumba Tengah', '87'),
  ('Kabupaten Sumba Barat', '87'), ('Kabupaten Sumba Barat Daya', '87'), ('Kabupaten Rote Ndao', '85'),
  ('Kabupaten Sabu Raijua', '85')
) AS cities(city_name, postal)
WHERE p.country_code = 'ID' AND p.name = 'Nusa Tenggara Timur'
ON CONFLICT DO NOTHING;

-- MALUKU
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'ID', city_name, postal FROM provinces p,
(VALUES
  ('Kota Ambon', '97'), ('Kota Tual', '97'), ('Kabupaten Maluku Tengah', '97'), ('Kabupaten Maluku Tenggara', '97'),
  ('Kabupaten Maluku Tenggara Barat', '97'), ('Kabupaten Buru', '97'), ('Kabupaten Buru Selatan', '97'),
  ('Kabupaten Kepulauan Aru', '97'), ('Kabupaten Seram Bagian Barat', '97'), ('Kabupaten Seram Bagian Timur', '97'),
  ('Kabupaten Kepulauan Tanimbar', '97')
) AS cities(city_name, postal)
WHERE p.country_code = 'ID' AND p.name = 'Maluku'
ON CONFLICT DO NOTHING;

-- MALUKU UTARA
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'ID', city_name, postal FROM provinces p,
(VALUES
  ('Kota Ternate', '97'), ('Kota Tidore Kepulauan', '97'), ('Kabupaten Halmahera Barat', '97'),
  ('Kabupaten Halmahera Tengah', '97'), ('Kabupaten Halmahera Utara', '97'), ('Kabupaten Halmahera Selatan', '97'),
  ('Kabupaten Halmahera Timur', '97'), ('Kabupaten Kepulauan Sula', '97'), ('Kabupaten Pulau Morotai', '97'),
  ('Kabupaten Pulau Taliabu', '97')
) AS cities(city_name, postal)
WHERE p.country_code = 'ID' AND p.name = 'Maluku Utara'
ON CONFLICT DO NOTHING;

-- PAPUA
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'ID', city_name, postal FROM provinces p,
(VALUES
  ('Kota Jayapura', '99'), ('Kabupaten Jayapura', '99'), ('Kabupaten Keerom', '99'), ('Kabupaten Sarmi', '99'),
  ('Kabupaten Mamberamo Raya', '99'), ('Kabupaten Kepulauan Yapen', '98'), ('Kabupaten Biak Numfor', '98'),
  ('Kabupaten Waropen', '98'), ('Kabupaten Supiori', '98')
) AS cities(city_name, postal)
WHERE p.country_code = 'ID' AND p.name = 'Papua'
ON CONFLICT DO NOTHING;

-- PAPUA BARAT
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'ID', city_name, postal FROM provinces p,
(VALUES
  ('Kota Manokwari', '98'), ('Kabupaten Manokwari', '98'), ('Kabupaten Manokwari Selatan', '98'),
  ('Kabupaten Pegunungan Arfak', '98')
) AS cities(city_name, postal)
WHERE p.country_code = 'ID' AND p.name = 'Papua Barat'
ON CONFLICT DO NOTHING;

-- PAPUA SELATAN
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'ID', city_name, postal FROM provinces p,
(VALUES
  ('Kabupaten Merauke', '99'), ('Kabupaten Boven Digoel', '99'), ('Kabupaten Mappi', '99'), ('Kabupaten Asmat', '99')
) AS cities(city_name, postal)
WHERE p.country_code = 'ID' AND p.name = 'Papua Selatan'
ON CONFLICT DO NOTHING;

-- PAPUA TENGAH
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'ID', city_name, postal FROM provinces p,
(VALUES
  ('Kabupaten Nabire', '98'), ('Kabupaten Paniai', '98'), ('Kabupaten Deiyai', '98'), ('Kabupaten Intan Jaya', '98'),
  ('Kabupaten Dogiyai', '98'), ('Kabupaten Puncak', '98'), ('Kabupaten Mimika', '99'), ('Kabupaten Puncak Jaya', '98')
) AS cities(city_name, postal)
WHERE p.country_code = 'ID' AND p.name = 'Papua Tengah'
ON CONFLICT DO NOTHING;

-- PAPUA PEGUNUNGAN
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'ID', city_name, postal FROM provinces p,
(VALUES
  ('Kabupaten Jayawijaya', '99'), ('Kabupaten Pegunungan Bintang', '99'), ('Kabupaten Yahukimo', '99'),
  ('Kabupaten Tolikara', '99'), ('Kabupaten Mamberamo Tengah', '99'), ('Kabupaten Yalimo', '99'),
  ('Kabupaten Lanny Jaya', '99'), ('Kabupaten Nduga', '99')
) AS cities(city_name, postal)
WHERE p.country_code = 'ID' AND p.name = 'Papua Pegunungan'
ON CONFLICT DO NOTHING;

-- PAPUA BARAT DAYA
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'ID', city_name, postal FROM provinces p,
(VALUES
  ('Kota Sorong', '98'), ('Kabupaten Sorong', '98'), ('Kabupaten Sorong Selatan', '98'), ('Kabupaten Raja Ampat', '98'),
  ('Kabupaten Tambrauw', '98'), ('Kabupaten Maybrat', '98'), ('Kabupaten Teluk Bintuni', '98'),
  ('Kabupaten Teluk Wondama', '98'), ('Kabupaten Fakfak', '98'), ('Kabupaten Kaimana', '98')
) AS cities(city_name, postal)
WHERE p.country_code = 'ID' AND p.name = 'Papua Barat Daya'
ON CONFLICT DO NOTHING;

-- ============================================
-- MALAYSIA - 16 STATES
-- ============================================

INSERT INTO provinces (country_code, name, code) VALUES
('MY', 'Johor', 'JHR'), ('MY', 'Kedah', 'KDH'), ('MY', 'Kelantan', 'KTN'), ('MY', 'Melaka', 'MLK'),
('MY', 'Negeri Sembilan', 'NSN'), ('MY', 'Pahang', 'PHG'), ('MY', 'Perak', 'PRK'), ('MY', 'Perlis', 'PLS'),
('MY', 'Penang', 'PNG'), ('MY', 'Sabah', 'SBH'), ('MY', 'Sarawak', 'SWK'), ('MY', 'Selangor', 'SGR'),
('MY', 'Terengganu', 'TRG'), ('MY', 'Kuala Lumpur', 'KUL'), ('MY', 'Labuan', 'LBN'), ('MY', 'Putrajaya', 'PJY')
ON CONFLICT DO NOTHING;

-- Malaysia Cities - Johor
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'MY', city_name, postal FROM provinces p,
(VALUES
  ('Johor Bahru', '80'), ('Iskandar Puteri', '79'), ('Pasir Gudang', '81'), ('Kulai', '81'),
  ('Kota Tinggi', '81'), ('Pontian', '82'), ('Batu Pahat', '83'), ('Muar', '84'),
  ('Kluang', '86'), ('Segamat', '85'), ('Mersing', '86'), ('Tangkak', '84')
) AS cities(city_name, postal)
WHERE p.country_code = 'MY' AND p.name = 'Johor'
ON CONFLICT DO NOTHING;

-- Malaysia Cities - Kedah
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'MY', city_name, postal FROM provinces p,
(VALUES
  ('Alor Setar', '05'), ('Sungai Petani', '08'), ('Kulim', '09'), ('Langkawi', '07'),
  ('Jitra', '06'), ('Kuala Kedah', '06'), ('Baling', '09'), ('Pendang', '06')
) AS cities(city_name, postal)
WHERE p.country_code = 'MY' AND p.name = 'Kedah'
ON CONFLICT DO NOTHING;

-- Malaysia Cities - Kelantan
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'MY', city_name, postal FROM provinces p,
(VALUES
  ('Kota Bharu', '15'), ('Pasir Mas', '17'), ('Tanah Merah', '17'), ('Machang', '18'),
  ('Kuala Krai', '18'), ('Gua Musang', '18'), ('Tumpat', '16'), ('Bachok', '16')
) AS cities(city_name, postal)
WHERE p.country_code = 'MY' AND p.name = 'Kelantan'
ON CONFLICT DO NOTHING;

-- Malaysia Cities - Melaka
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'MY', city_name, postal FROM provinces p,
(VALUES
  ('Melaka City', '75'), ('Ayer Keroh', '75'), ('Alor Gajah', '78'), ('Jasin', '77'), ('Masjid Tanah', '78')
) AS cities(city_name, postal)
WHERE p.country_code = 'MY' AND p.name = 'Melaka'
ON CONFLICT DO NOTHING;

-- Malaysia Cities - Negeri Sembilan
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'MY', city_name, postal FROM provinces p,
(VALUES
  ('Seremban', '70'), ('Port Dickson', '71'), ('Nilai', '71'), ('Bahau', '72'),
  ('Kuala Pilah', '72'), ('Tampin', '73'), ('Rembau', '71')
) AS cities(city_name, postal)
WHERE p.country_code = 'MY' AND p.name = 'Negeri Sembilan'
ON CONFLICT DO NOTHING;

-- Malaysia Cities - Pahang
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'MY', city_name, postal FROM provinces p,
(VALUES
  ('Kuantan', '25'), ('Temerloh', '28'), ('Bentong', '28'), ('Raub', '27'), ('Jerantut', '27'),
  ('Pekan', '26'), ('Rompin', '26'), ('Cameron Highlands', '39'), ('Genting Highlands', '69')
) AS cities(city_name, postal)
WHERE p.country_code = 'MY' AND p.name = 'Pahang'
ON CONFLICT DO NOTHING;

-- Malaysia Cities - Perak
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'MY', city_name, postal FROM provinces p,
(VALUES
  ('Ipoh', '30'), ('Taiping', '34'), ('Teluk Intan', '36'), ('Lumut', '32'), ('Manjung', '32'),
  ('Kuala Kangsar', '33'), ('Kampar', '31'), ('Batu Gajah', '31'), ('Tanjung Malim', '35'), ('Gerik', '33')
) AS cities(city_name, postal)
WHERE p.country_code = 'MY' AND p.name = 'Perak'
ON CONFLICT DO NOTHING;

-- Malaysia Cities - Perlis
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'MY', city_name, postal FROM provinces p,
(VALUES ('Kangar', '01'), ('Arau', '02'), ('Padang Besar', '01')) AS cities(city_name, postal)
WHERE p.country_code = 'MY' AND p.name = 'Perlis'
ON CONFLICT DO NOTHING;

-- Malaysia Cities - Penang
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'MY', city_name, postal FROM provinces p,
(VALUES
  ('George Town', '10'), ('Butterworth', '12'), ('Bukit Mertajam', '14'), ('Nibong Tebal', '14'),
  ('Bayan Lepas', '11'), ('Balik Pulau', '11'), ('Kepala Batas', '13')
) AS cities(city_name, postal)
WHERE p.country_code = 'MY' AND p.name = 'Penang'
ON CONFLICT DO NOTHING;

-- Malaysia Cities - Sabah
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'MY', city_name, postal FROM provinces p,
(VALUES
  ('Kota Kinabalu', '88'), ('Sandakan', '90'), ('Tawau', '91'), ('Lahad Datu', '91'), ('Keningau', '89'),
  ('Semporna', '91'), ('Kudat', '89'), ('Beaufort', '89'), ('Papar', '89'), ('Ranau', '89')
) AS cities(city_name, postal)
WHERE p.country_code = 'MY' AND p.name = 'Sabah'
ON CONFLICT DO NOTHING;

-- Malaysia Cities - Sarawak
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'MY', city_name, postal FROM provinces p,
(VALUES
  ('Kuching', '93'), ('Miri', '98'), ('Sibu', '96'), ('Bintulu', '97'), ('Limbang', '98'),
  ('Sarikei', '96'), ('Sri Aman', '95'), ('Kapit', '96'), ('Mukah', '96'), ('Lawas', '98')
) AS cities(city_name, postal)
WHERE p.country_code = 'MY' AND p.name = 'Sarawak'
ON CONFLICT DO NOTHING;

-- Malaysia Cities - Selangor
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'MY', city_name, postal FROM provinces p,
(VALUES
  ('Shah Alam', '40'), ('Petaling Jaya', '46'), ('Subang Jaya', '47'), ('Klang', '41'), ('Ampang', '68'),
  ('Kajang', '43'), ('Selayang', '68'), ('Rawang', '48'), ('Sepang', '43'), ('Kuala Selangor', '45'),
  ('Sabak Bernam', '45'), ('Hulu Langat', '43'), ('Gombak', '68'), ('Puchong', '47'),
  ('Cyberjaya', '63'), ('Bangi', '43')
) AS cities(city_name, postal)
WHERE p.country_code = 'MY' AND p.name = 'Selangor'
ON CONFLICT DO NOTHING;

-- Malaysia Cities - Terengganu
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'MY', city_name, postal FROM provinces p,
(VALUES
  ('Kuala Terengganu', '20'), ('Kemaman', '24'), ('Dungun', '23'), ('Besut', '22'),
  ('Marang', '21'), ('Hulu Terengganu', '21'), ('Setiu', '22')
) AS cities(city_name, postal)
WHERE p.country_code = 'MY' AND p.name = 'Terengganu'
ON CONFLICT DO NOTHING;

-- Malaysia Cities - Kuala Lumpur
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'MY', city_name, postal FROM provinces p,
(VALUES
  ('Kuala Lumpur City Centre', '50'), ('Bukit Bintang', '55'), ('Cheras', '56'), ('Kepong', '52'),
  ('Setapak', '53'), ('Bangsar', '59'), ('Mont Kiara', '50'), ('Wangsa Maju', '53'),
  ('Titiwangsa', '53'), ('Sentul', '51')
) AS cities(city_name, postal)
WHERE p.country_code = 'MY' AND p.name = 'Kuala Lumpur'
ON CONFLICT DO NOTHING;

-- Malaysia Cities - Labuan
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'MY', city_name, postal FROM provinces p,
(VALUES ('Labuan Town', '87')) AS cities(city_name, postal)
WHERE p.country_code = 'MY' AND p.name = 'Labuan'
ON CONFLICT DO NOTHING;

-- Malaysia Cities - Putrajaya
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'MY', city_name, postal FROM provinces p,
(VALUES ('Putrajaya', '62')) AS cities(city_name, postal)
WHERE p.country_code = 'MY' AND p.name = 'Putrajaya'
ON CONFLICT DO NOTHING;

-- ============================================
-- SINGAPORE - 5 REGIONS
-- ============================================

INSERT INTO provinces (country_code, name, code) VALUES
('SG', 'Central Region', 'CR'), ('SG', 'East Region', 'ER'), ('SG', 'North Region', 'NR'),
('SG', 'North-East Region', 'NER'), ('SG', 'West Region', 'WR')
ON CONFLICT DO NOTHING;

-- Singapore Cities - Central Region
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'SG', city_name, postal FROM provinces p,
(VALUES
  ('Bishan', '57'), ('Bukit Merah', '15'), ('Bukit Timah', '58'), ('Downtown Core', '01'),
  ('Geylang', '38'), ('Kallang', '33'), ('Marine Parade', '44'), ('Museum', '17'),
  ('Newton', '22'), ('Novena', '30'), ('Orchard', '23'), ('Outram', '16'),
  ('Queenstown', '14'), ('River Valley', '23'), ('Rochor', '18'), ('Singapore River', '04'),
  ('Southern Islands', '09'), ('Tanglin', '24'), ('Toa Payoh', '31')
) AS cities(city_name, postal)
WHERE p.country_code = 'SG' AND p.name = 'Central Region'
ON CONFLICT DO NOTHING;

-- Singapore Cities - East Region
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'SG', city_name, postal FROM provinces p,
(VALUES
  ('Bedok', '46'), ('Changi', '49'), ('Changi Bay', '48'), ('Pasir Ris', '51'),
  ('Paya Lebar', '40'), ('Tampines', '52')
) AS cities(city_name, postal)
WHERE p.country_code = 'SG' AND p.name = 'East Region'
ON CONFLICT DO NOTHING;

-- Singapore Cities - North Region
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'SG', city_name, postal FROM provinces p,
(VALUES
  ('Central Water Catchment', '78'), ('Lim Chu Kang', '71'), ('Mandai', '72'), ('Sembawang', '75'),
  ('Simpang', '78'), ('Sungei Kadut', '72'), ('Woodlands', '73'), ('Yishun', '76')
) AS cities(city_name, postal)
WHERE p.country_code = 'SG' AND p.name = 'North Region'
ON CONFLICT DO NOTHING;

-- Singapore Cities - North-East Region
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'SG', city_name, postal FROM provinces p,
(VALUES
  ('Ang Mo Kio', '56'), ('Hougang', '53'), ('North-Eastern Islands', '50'), ('Punggol', '82'),
  ('Seletar', '79'), ('Sengkang', '54'), ('Serangoon', '55')
) AS cities(city_name, postal)
WHERE p.country_code = 'SG' AND p.name = 'North-East Region'
ON CONFLICT DO NOTHING;

-- Singapore Cities - West Region
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'SG', city_name, postal FROM provinces p,
(VALUES
  ('Boon Lay', '61'), ('Bukit Batok', '65'), ('Bukit Panjang', '67'), ('Choa Chu Kang', '68'),
  ('Clementi', '12'), ('Jurong East', '60'), ('Jurong West', '64'), ('Pioneer', '62'),
  ('Tengah', '69'), ('Tuas', '63'), ('Western Islands', '62'), ('Western Water Catchment', '71')
) AS cities(city_name, postal)
WHERE p.country_code = 'SG' AND p.name = 'West Region'
ON CONFLICT DO NOTHING;

-- ============================================
-- THAILAND - 20 MAJOR PROVINCES
-- ============================================

INSERT INTO provinces (country_code, name, code) VALUES
('TH', 'Bangkok', 'BKK'), ('TH', 'Chiang Mai', 'CNX'), ('TH', 'Chiang Rai', 'CEI'), ('TH', 'Phuket', 'HKT'),
('TH', 'Chonburi', 'CBI'), ('TH', 'Nonthaburi', 'NBI'), ('TH', 'Pathum Thani', 'PTN'), ('TH', 'Samut Prakan', 'SPK'),
('TH', 'Nakhon Ratchasima', 'NMA'), ('TH', 'Khon Kaen', 'KKN'), ('TH', 'Udon Thani', 'UDN'), ('TH', 'Songkhla', 'SKA'),
('TH', 'Surat Thani', 'SNI'), ('TH', 'Krabi', 'KBI'), ('TH', 'Ayutthaya', 'AYA'), ('TH', 'Rayong', 'RYG'),
('TH', 'Nakhon Si Thammarat', 'NST'), ('TH', 'Ubon Ratchathani', 'UBP'), ('TH', 'Lampang', 'LPG'), ('TH', 'Phitsanulok', 'PLK')
ON CONFLICT DO NOTHING;

-- Thailand Cities - Bangkok (50 Districts)
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'TH', city_name, postal FROM provinces p,
(VALUES
  ('Phra Nakhon', '10'), ('Dusit', '10'), ('Nong Chok', '10'), ('Bang Rak', '10'),
  ('Bang Khen', '10'), ('Bang Kapi', '10'), ('Pathum Wan', '10'), ('Pom Prap Sattru Phai', '10'),
  ('Phra Khanong', '10'), ('Min Buri', '10'), ('Lat Krabang', '10'), ('Yan Nawa', '10'),
  ('Samphanthawong', '10'), ('Phaya Thai', '10'), ('Thon Buri', '10'), ('Bangkok Yai', '10'),
  ('Huai Khwang', '10'), ('Khlong San', '10'), ('Taling Chan', '10'), ('Bangkok Noi', '10'),
  ('Bang Khun Thian', '10'), ('Phasi Charoen', '10'), ('Nong Khaem', '10'), ('Rat Burana', '10'),
  ('Bang Phlat', '10'), ('Din Daeng', '10'), ('Bueng Kum', '10'), ('Sathon', '10'),
  ('Bang Sue', '10'), ('Chatuchak', '10'), ('Bang Kho Laem', '10'), ('Prawet', '10'),
  ('Khlong Toei', '10'), ('Suan Luang', '10'), ('Chom Thong', '10'), ('Don Mueang', '10'),
  ('Ratchathewi', '10'), ('Lat Phrao', '10'), ('Watthana', '10'), ('Bang Khae', '10'),
  ('Lak Si', '10'), ('Sai Mai', '10'), ('Khan Na Yao', '10'), ('Saphan Sung', '10'),
  ('Wang Thonglang', '10'), ('Khlong Sam Wa', '10'), ('Bang Na', '10'), ('Thawi Watthana', '10'),
  ('Thung Khru', '10'), ('Bang Bon', '10')
) AS cities(city_name, postal)
WHERE p.country_code = 'TH' AND p.name = 'Bangkok'
ON CONFLICT DO NOTHING;

-- Thailand Cities - Chiang Mai
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'TH', city_name, postal FROM provinces p,
(VALUES
  ('Mueang Chiang Mai', '50'), ('Chom Thong', '50'), ('Mae Chaem', '50'), ('Chiang Dao', '50'),
  ('Doi Saket', '50'), ('Mae Taeng', '50'), ('Mae Rim', '50'), ('Samoeng', '50'),
  ('Fang', '50'), ('Mae Ai', '50'), ('Phrao', '50'), ('San Pa Tong', '50'),
  ('San Kamphaeng', '50'), ('San Sai', '50'), ('Hang Dong', '50'), ('Hot', '50'),
  ('Doi Tao', '50'), ('Omkoi', '50'), ('Saraphi', '50'), ('Wiang Haeng', '50')
) AS cities(city_name, postal)
WHERE p.country_code = 'TH' AND p.name = 'Chiang Mai'
ON CONFLICT DO NOTHING;

-- Thailand Cities - Phuket
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'TH', city_name, postal FROM provinces p,
(VALUES ('Mueang Phuket', '83'), ('Kathu', '83'), ('Thalang', '83')) AS cities(city_name, postal)
WHERE p.country_code = 'TH' AND p.name = 'Phuket'
ON CONFLICT DO NOTHING;

-- Thailand Cities - Chonburi
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'TH', city_name, postal FROM provinces p,
(VALUES
  ('Mueang Chonburi', '20'), ('Ban Bueng', '20'), ('Nong Yai', '20'), ('Bang Lamung', '20'),
  ('Phan Thong', '20'), ('Phanat Nikhom', '20'), ('Si Racha', '20'), ('Ko Sichang', '20'),
  ('Sattahip', '20'), ('Bo Thong', '20'), ('Ko Chan', '20')
) AS cities(city_name, postal)
WHERE p.country_code = 'TH' AND p.name = 'Chonburi'
ON CONFLICT DO NOTHING;

-- Thailand Cities - Nonthaburi
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'TH', city_name, postal FROM provinces p,
(VALUES
  ('Mueang Nonthaburi', '11'), ('Bang Kruai', '11'), ('Bang Yai', '11'),
  ('Bang Bua Thong', '11'), ('Sai Noi', '11'), ('Pak Kret', '11')
) AS cities(city_name, postal)
WHERE p.country_code = 'TH' AND p.name = 'Nonthaburi'
ON CONFLICT DO NOTHING;

-- ============================================
-- VIETNAM - 20 MAJOR PROVINCES
-- ============================================

INSERT INTO provinces (country_code, name, code) VALUES
('VN', 'Ho Chi Minh City', 'SG'), ('VN', 'Hanoi', 'HN'), ('VN', 'Da Nang', 'DN'),
('VN', 'Hai Phong', 'HP'), ('VN', 'Can Tho', 'CT'), ('VN', 'Binh Duong', 'BD'),
('VN', 'Dong Nai', 'DN2'), ('VN', 'Khanh Hoa', 'KH'), ('VN', 'Quang Ninh', 'QN'),
('VN', 'Lam Dong', 'LD'), ('VN', 'Thua Thien Hue', 'TTH'), ('VN', 'Ba Ria Vung Tau', 'VT'),
('VN', 'Nghe An', 'NA'), ('VN', 'Thanh Hoa', 'TH'), ('VN', 'Binh Thuan', 'BTN'),
('VN', 'Long An', 'LA'), ('VN', 'Tien Giang', 'TG'), ('VN', 'An Giang', 'AG'),
('VN', 'Quang Nam', 'QNM'), ('VN', 'Gia Lai', 'GL')
ON CONFLICT DO NOTHING;

-- Vietnam Cities - Ho Chi Minh City
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'VN', city_name, postal FROM provinces p,
(VALUES
  ('District 1', '70'), ('District 2', '70'), ('District 3', '70'), ('District 4', '70'),
  ('District 5', '70'), ('District 6', '70'), ('District 7', '70'), ('District 8', '70'),
  ('District 9', '70'), ('District 10', '70'), ('District 11', '70'), ('District 12', '70'),
  ('Binh Tan', '70'), ('Binh Thanh', '70'), ('Go Vap', '70'), ('Phu Nhuan', '70'),
  ('Tan Binh', '70'), ('Tan Phu', '70'), ('Thu Duc City', '70'), ('Binh Chanh', '70'),
  ('Can Gio', '70'), ('Cu Chi', '70'), ('Hoc Mon', '70'), ('Nha Be', '70')
) AS cities(city_name, postal)
WHERE p.country_code = 'VN' AND p.name = 'Ho Chi Minh City'
ON CONFLICT DO NOTHING;

-- Vietnam Cities - Hanoi
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'VN', city_name, postal FROM provinces p,
(VALUES
  ('Ba Dinh', '10'), ('Hoan Kiem', '10'), ('Tay Ho', '10'), ('Long Bien', '10'),
  ('Cau Giay', '10'), ('Dong Da', '10'), ('Hai Ba Trung', '10'), ('Hoang Mai', '10'),
  ('Thanh Xuan', '10'), ('Bac Tu Liem', '10'), ('Nam Tu Liem', '10'), ('Ha Dong', '10'),
  ('Son Tay', '10'), ('Ba Vi', '10'), ('Chuong My', '10'), ('Dan Phuong', '10'),
  ('Dong Anh', '10'), ('Gia Lam', '10'), ('Hoai Duc', '10'), ('Me Linh', '10')
) AS cities(city_name, postal)
WHERE p.country_code = 'VN' AND p.name = 'Hanoi'
ON CONFLICT DO NOTHING;

-- Vietnam Cities - Da Nang
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'VN', city_name, postal FROM provinces p,
(VALUES
  ('Hai Chau', '55'), ('Thanh Khe', '55'), ('Son Tra', '55'), ('Ngu Hanh Son', '55'),
  ('Lien Chieu', '55'), ('Cam Le', '55'), ('Hoa Vang', '55'), ('Hoang Sa', '55')
) AS cities(city_name, postal)
WHERE p.country_code = 'VN' AND p.name = 'Da Nang'
ON CONFLICT DO NOTHING;

-- ============================================
-- PHILIPPINES - 20 MAJOR PROVINCES
-- ============================================

INSERT INTO provinces (country_code, name, code) VALUES
('PH', 'Metro Manila', 'NCR'), ('PH', 'Cebu', 'CEB'), ('PH', 'Davao del Sur', 'DAS'),
('PH', 'Cavite', 'CAV'), ('PH', 'Laguna', 'LAG'), ('PH', 'Bulacan', 'BUL'),
('PH', 'Rizal', 'RIZ'), ('PH', 'Pampanga', 'PAM'), ('PH', 'Batangas', 'BTG'),
('PH', 'Pangasinan', 'PAN'), ('PH', 'Negros Occidental', 'NEC'), ('PH', 'Iloilo', 'ILI'),
('PH', 'Zambales', 'ZMB'), ('PH', 'Nueva Ecija', 'NUE'), ('PH', 'Quezon', 'QUE'),
('PH', 'Tarlac', 'TAR'), ('PH', 'Bohol', 'BOH'), ('PH', 'Leyte', 'LEY'),
('PH', 'Albay', 'ALB'), ('PH', 'Palawan', 'PLW')
ON CONFLICT DO NOTHING;

-- Philippines Cities - Metro Manila
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'PH', city_name, postal FROM provinces p,
(VALUES
  ('Manila', '10'), ('Quezon City', '11'), ('Makati', '12'), ('Pasig', '16'),
  ('Taguig', '16'), ('Mandaluyong', '15'), ('Pasay', '13'), ('Caloocan', '14'),
  ('Las Pinas', '17'), ('Muntinlupa', '17'), ('Paranaque', '17'), ('Valenzuela', '14'),
  ('Malabon', '14'), ('Navotas', '14'), ('San Juan', '15'), ('Marikina', '18'), ('Pateros', '16')
) AS cities(city_name, postal)
WHERE p.country_code = 'PH' AND p.name = 'Metro Manila'
ON CONFLICT DO NOTHING;

-- Philippines Cities - Cebu
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'PH', city_name, postal FROM provinces p,
(VALUES
  ('Cebu City', '60'), ('Mandaue', '60'), ('Lapu-Lapu', '60'), ('Talisay', '60'),
  ('Danao', '60'), ('Carcar', '60'), ('Naga', '60'), ('Toledo', '60'),
  ('Bogo', '60'), ('Minglanilla', '60'), ('Consolacion', '60'), ('Liloan', '60'),
  ('Cordova', '60'), ('Compostela', '60')
) AS cities(city_name, postal)
WHERE p.country_code = 'PH' AND p.name = 'Cebu'
ON CONFLICT DO NOTHING;

-- Philippines Cities - Davao del Sur
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'PH', city_name, postal FROM provinces p,
(VALUES
  ('Davao City', '80'), ('Digos', '80'), ('Bansalan', '80'), ('Hagonoy', '80'),
  ('Kiblawan', '80'), ('Magsaysay', '80'), ('Malalag', '80'), ('Matanao', '80'),
  ('Padada', '80'), ('Santa Cruz', '80'), ('Sulop', '80')
) AS cities(city_name, postal)
WHERE p.country_code = 'PH' AND p.name = 'Davao del Sur'
ON CONFLICT DO NOTHING;

-- Philippines Cities - Cavite
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'PH', city_name, postal FROM provinces p,
(VALUES
  ('Bacoor', '41'), ('Imus', '41'), ('Dasmarinas', '41'), ('General Trias', '41'),
  ('Cavite City', '41'), ('Tagaytay', '41'), ('Trece Martires', '41'), ('Silang', '41'),
  ('Carmona', '41'), ('General Mariano Alvarez', '41'), ('Kawit', '41'),
  ('Noveleta', '41'), ('Rosario', '41'), ('Tanza', '41')
) AS cities(city_name, postal)
WHERE p.country_code = 'PH' AND p.name = 'Cavite'
ON CONFLICT DO NOTHING;

-- Philippines Cities - Laguna
INSERT INTO cities (province_id, country_code, name, postal_code_prefix)
SELECT p.id, 'PH', city_name, postal FROM provinces p,
(VALUES
  ('Santa Rosa', '40'), ('Calamba', '40'), ('Binan', '40'), ('San Pedro', '40'),
  ('Cabuyao', '40'), ('San Pablo', '40'), ('Los Banos', '40'), ('Bay', '40'),
  ('Calauan', '40'), ('Alaminos', '40'), ('Nagcarlan', '40'), ('Liliw', '40'),
  ('Pagsanjan', '40'), ('Pila', '40'), ('Victoria', '40')
) AS cities(city_name, postal)
WHERE p.country_code = 'PH' AND p.name = 'Laguna'
ON CONFLICT DO NOTHING;

-- ============================================


-- +goose Down
SELECT 1; -- Seed rollback is not supported. Use docker compose down -v for full reset.
