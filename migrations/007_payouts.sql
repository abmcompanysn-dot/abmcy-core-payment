-- Migration 007 : versements sortants (payouts).
--
-- Une app (ou l'admin) envoie de l'argent vers un numéro mobile money.
-- C'est un `payments` de type 'payout' : mêmes colonnes de suivi (statut,
-- frais, relais), plus le destinataire (téléphone + opérateur + pays).
--
-- La commission 50 F/$ s'applique aussi ici : fee_cfa est prélevé, net_cfa
-- est le montant RÉELLEMENT envoyé au destinataire (amount_cfa - fee_cfa).

ALTER TABLE payments
	ADD COLUMN IF NOT EXISTS recipient_phone    TEXT,
	ADD COLUMN IF NOT EXISTS recipient_operator TEXT,
	ADD COLUMN IF NOT EXISTS country            TEXT;

CREATE INDEX IF NOT EXISTS idx_payments_type_created
	ON payments (type, created_at DESC);
