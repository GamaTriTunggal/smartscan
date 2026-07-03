-- +goose Up
-- Reference data: certification bodies and social platforms.
-- Factual, publicly documented information about official certification
-- authorities in ASEAN countries, plus well-known social/commerce platforms.


-- Indonesia Certifications
INSERT INTO certification_types (country_code, code, name, description, logo_url, website_url, display_order) VALUES
('ID', 'BPOM', 'BPOM', 'Badan Pengawas Obat dan Makanan - Indonesian FDA for food, drugs, and cosmetics', '/logos/certs/bpom.png', 'https://www.pom.go.id', 1),
('ID', 'HALAL_BPJPH', 'Halal BPJPH', 'Badan Penyelenggara Jaminan Produk Halal - Official Indonesian Halal Certification', '/logos/certs/halal-bpjph.png', 'https://www.halal.go.id', 2),
('ID', 'HALAL_MUI', 'Halal MUI', 'Majelis Ulama Indonesia - Legacy Halal Certification', '/logos/certs/halal-mui.png', 'https://www.halalmui.org', 3),
('ID', 'SNI', 'SNI', 'Standar Nasional Indonesia - Indonesian National Standard', '/logos/certs/sni.png', 'https://www.bsn.go.id', 4),
('ID', 'PIRT', 'PIRT', 'Pangan Industri Rumah Tangga - Home Industry Food Registration', '/logos/certs/pirt.png', NULL, 5),
('ID', 'ORGANIC_ID', 'Organik Indonesia', 'Indonesian Organic Certification', '/logos/certs/organic-id.png', NULL, 6);

-- Malaysia Certifications
INSERT INTO certification_types (country_code, code, name, description, logo_url, website_url, display_order) VALUES
('MY', 'JAKIM_HALAL', 'JAKIM Halal', 'Jabatan Kemajuan Islam Malaysia - Malaysian Halal Certification', '/logos/certs/jakim.png', 'https://www.halal.gov.my', 1),
('MY', 'KKM', 'KKM', 'Kementerian Kesihatan Malaysia - Ministry of Health Malaysia', '/logos/certs/kkm.png', 'https://www.moh.gov.my', 2),
('MY', 'SIRIM', 'SIRIM', 'Standards and Industrial Research Institute of Malaysia', '/logos/certs/sirim.png', 'https://www.sirim.my', 3),
('MY', 'MESTI', 'MeSTI', 'Makanan Selamat Tanggungjawab Industri - Food Safety Certification', '/logos/certs/mesti.png', NULL, 4),
('MY', 'MYORGANIC', 'MyOrganic', 'Malaysian Organic Certification Scheme', '/logos/certs/myorganic.png', NULL, 5);

-- Philippines Certifications
INSERT INTO certification_types (country_code, code, name, description, logo_url, website_url, display_order) VALUES
('PH', 'FDA_PH', 'FDA Philippines', 'Food and Drug Administration Philippines', '/logos/certs/fda-ph.png', 'https://www.fda.gov.ph', 1),
('PH', 'HALAL_IDCP', 'Halal IDCP', 'Islamic Da''wah Council of the Philippines Halal Certification', '/logos/certs/halal-idcp.png', NULL, 2),
('PH', 'BPS', 'BPS', 'Bureau of Philippine Standards', '/logos/certs/bps.png', 'https://www.bps.dti.gov.ph', 3),
('PH', 'ORGANIC_PH', 'Organic Philippines', 'Philippine National Organic Agriculture Program', '/logos/certs/organic-ph.png', NULL, 4);

-- Singapore Certifications
INSERT INTO certification_types (country_code, code, name, description, logo_url, website_url, display_order) VALUES
('SG', 'SFA', 'SFA', 'Singapore Food Agency', '/logos/certs/sfa.png', 'https://www.sfa.gov.sg', 1),
('SG', 'MUIS_HALAL', 'MUIS Halal', 'Majlis Ugama Islam Singapura - Singapore Halal Certification', '/logos/certs/muis.png', 'https://www.muis.gov.sg', 2),
('SG', 'SS_MARK', 'Singapore Standards', 'Singapore Standards Mark', '/logos/certs/ss-mark.png', 'https://www.enterprisesg.gov.sg', 3),
('SG', 'HSA', 'HSA', 'Health Sciences Authority Singapore', '/logos/certs/hsa.png', 'https://www.hsa.gov.sg', 4);

-- Thailand Certifications
INSERT INTO certification_types (country_code, code, name, description, logo_url, website_url, display_order) VALUES
('TH', 'FDA_TH', 'อย. (FDA Thailand)', 'Thai Food and Drug Administration', '/logos/certs/fda-th.png', 'https://www.fda.moph.go.th', 1),
('TH', 'HALAL_CICOT', 'Halal CICOT', 'Central Islamic Council of Thailand Halal Certification', '/logos/certs/halal-cicot.png', NULL, 2),
('TH', 'TISI', 'TISI', 'Thai Industrial Standards Institute', '/logos/certs/tisi.png', 'https://www.tisi.go.th', 3),
('TH', 'ORGANIC_TH', 'Organic Thailand', 'Thai Organic Certification', '/logos/certs/organic-th.png', NULL, 4);

-- Vietnam Certifications
INSERT INTO certification_types (country_code, code, name, description, logo_url, website_url, display_order) VALUES
('VN', 'VFA', 'VFA', 'Vietnam Food Administration', '/logos/certs/vfa.png', NULL, 1),
('VN', 'HALAL_VN', 'Halal Vietnam', 'Vietnam Halal Certification', '/logos/certs/halal-vn.png', NULL, 2),
('VN', 'TCVN', 'TCVN', 'Tiêu chuẩn Việt Nam - Vietnamese National Standards', '/logos/certs/tcvn.png', NULL, 3),
('VN', 'VIETGAP', 'VietGAP', 'Vietnamese Good Agricultural Practices', '/logos/certs/vietgap.png', NULL, 4);

-- International Certifications (country_code = NULL)
INSERT INTO certification_types (country_code, code, name, description, logo_url, website_url, display_order) VALUES
(NULL, 'ISO_9001', 'ISO 9001', 'Quality Management System', '/logos/certs/iso-9001.png', 'https://www.iso.org', 1),
(NULL, 'ISO_14001', 'ISO 14001', 'Environmental Management System', '/logos/certs/iso-14001.png', 'https://www.iso.org', 2),
(NULL, 'ISO_22000', 'ISO 22000', 'Food Safety Management System', '/logos/certs/iso-22000.png', 'https://www.iso.org', 3),
(NULL, 'ISO_45001', 'ISO 45001', 'Occupational Health and Safety', '/logos/certs/iso-45001.png', 'https://www.iso.org', 4),
(NULL, 'HACCP', 'HACCP', 'Hazard Analysis Critical Control Points', '/logos/certs/haccp.png', NULL, 5),
(NULL, 'GMP', 'GMP', 'Good Manufacturing Practice', '/logos/certs/gmp.png', NULL, 6),
(NULL, 'USDA_ORGANIC', 'USDA Organic', 'United States Department of Agriculture Organic', '/logos/certs/usda-organic.png', 'https://www.usda.gov', 7),
(NULL, 'EU_ORGANIC', 'EU Organic', 'European Union Organic Certification', '/logos/certs/eu-organic.png', NULL, 8),
(NULL, 'FAIRTRADE', 'Fairtrade', 'Fairtrade International Certification', '/logos/certs/fairtrade.png', 'https://www.fairtrade.net', 9),
(NULL, 'NON_GMO', 'Non-GMO', 'Non-GMO Project Verified', '/logos/certs/non-gmo.png', 'https://www.nongmoproject.org', 10),
(NULL, 'VEGAN', 'Vegan Certified', 'Vegan Certification', '/logos/certs/vegan.png', NULL, 11),
(NULL, 'KOSHER', 'Kosher', 'Kosher Certification', '/logos/certs/kosher.png', NULL, 12),
(NULL, 'RAINFOREST', 'Rainforest Alliance', 'Rainforest Alliance Certified', '/logos/certs/rainforest.png', 'https://www.rainforest-alliance.org', 13),
(NULL, 'FSC', 'FSC', 'Forest Stewardship Council', '/logos/certs/fsc.png', 'https://www.fsc.org', 14),
(NULL, 'BRC', 'BRC', 'British Retail Consortium Global Standard', '/logos/certs/brc.png', 'https://www.brcgs.com', 15);

-- ============================================
-- SECTION 16: Social Media Platforms (Master List)
-- ============================================

INSERT INTO social_media_platforms (code, name, icon, base_url, deep_link_pattern, placeholder_text, validation_type, display_order) VALUES
-- Global Major Platforms
('instagram', 'Instagram', 'instagram', 'https://instagram.com/', 'https://instagram.com/{handle}', 'username (without @)', 'username', 1),
('tiktok', 'TikTok', 'tiktok', 'https://tiktok.com/@', 'https://tiktok.com/@{handle}', 'username (without @)', 'username', 2),
('whatsapp', 'WhatsApp', 'whatsapp', 'https://wa.me/', 'https://wa.me/{handle}', '+62812345678 (with country code)', 'phone', 3),
('facebook', 'Facebook', 'facebook', 'https://facebook.com/', 'https://facebook.com/{handle}', 'Page name or URL', 'text', 4),
('youtube', 'YouTube', 'youtube', 'https://youtube.com/@', 'https://youtube.com/@{handle}', 'channel name (without @)', 'username', 5),
('twitter', 'X (Twitter)', 'twitter', 'https://x.com/', 'https://x.com/{handle}', 'username (without @)', 'username', 6),
('telegram', 'Telegram', 'telegram', 'https://t.me/', 'https://t.me/{handle}', 'username (without @)', 'username', 7),
('linkedin', 'LinkedIn', 'linkedin', 'https://linkedin.com/', 'https://linkedin.com/{handle}', 'Company page URL', 'text', 8),
-- Contact
('email', 'Email', 'mail', 'mailto:', 'mailto:{handle}', 'contact@company.com', 'email', 9),
-- Regional Platforms
('line', 'LINE', 'line', 'https://line.me/', 'https://line.me/R/ti/p/{handle}', 'username (without @)', 'username', 10),
('wechat', 'WeChat', 'wechat', NULL, NULL, 'WeChat ID', 'text', 11),
('zalo', 'Zalo', 'zalo', 'https://zalo.me/', 'https://zalo.me/{handle}', '+84xxx (with country code)', 'phone', 12),
-- E-commerce Platforms
('shopee', 'Shopee', 'shopee', 'https://shopee.co.id/', 'https://shopee.co.id/{handle}', 'store name (e.g., your_store)', 'text', 13),
('tokopedia', 'Tokopedia', 'tokopedia', 'https://tokopedia.com/', 'https://tokopedia.com/{handle}', 'store name (e.g., your-store)', 'text', 14),
('lazada', 'Lazada', 'lazada', 'https://lazada.co.id/', 'https://lazada.co.id/{handle}', 'store name (e.g., official-store)', 'text', 15),
('blibli', 'Blibli', 'blibli', 'https://www.blibli.com/merchant/', 'https://www.blibli.com/merchant/{handle}', 'store name (e.g., official-store)', 'text', 16);

-- Default branding (monochrome). Admins can change this under Admin → Appearance.
INSERT INTO app_settings (setting_key, setting_value) VALUES
('branding', '{
  "app_name": "smartscan",
  "logo_url": "",
  "header_gradient_start": "#0a0a0a",
  "header_gradient_end": "#27272a",
  "header_text_color": "#ffffff",
  "button_bg_color": "#18181b",
  "button_text_color": "#ffffff"
}'::jsonb)
ON CONFLICT (setting_key) DO NOTHING;

-- +goose Down
DELETE FROM certification_types;
DELETE FROM social_media_platforms;
DELETE FROM app_settings WHERE setting_key = 'branding';
