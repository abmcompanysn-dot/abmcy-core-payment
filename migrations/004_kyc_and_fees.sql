-- Migration 004 : niveau KYC des apps + commission ABMCY Core sur les paiements.
--
-- KYC : sans vérification, une app ne peut encaisser que jusqu'à 200 000 FCFA
-- par transaction. Après KYC (validé à la main), la limite passe à 1 000 000.
-- Les seuils vivent dans le code (config), seul le NIVEAU est stocké ici.
--
-- Commission : ABMCY Core prélève sa marge sur chaque transaction — 50 FCFA
-- par dollar de transaction (taux configurable, défaut 600 F/USD). Retenue
-- sur le montant reversé au marchand. On fige le montant calculé au moment
-- de la création du paiement (le taux peut bouger ensuite).

ALTER TABLE apps
	ADD COLUMN IF NOT EXISTS kyc_level TEXT NOT NULL DEFAULT 'none'
		CHECK (kyc_level IN ('none', 'verified'));

ALTER TABLE payments
	ADD COLUMN IF NOT EXISTS fee_cfa       INTEGER NOT NULL DEFAULT 0,
	ADD COLUMN IF NOT EXISTS net_cfa       INTEGER,          -- amount_cfa - fee_cfa (reversé au marchand)
	ADD COLUMN IF NOT EXISTS usd_rate_used INTEGER;          -- taux F/USD figé au calcul
