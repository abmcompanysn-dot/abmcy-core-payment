-- Migration 005 : demandes d'inscription depuis la landing page publique.
--
-- Un visiteur remplit le formulaire "créer ma passerelle" -> une ligne ici
-- en statut 'pending'. L'admin (back-office) approuve -> une app est créée
-- (table apps) et les clés envoyées par email ; ou rejette avec un motif.
-- On garde la demande même après traitement (audit / renvoi de clés).

CREATE TABLE IF NOT EXISTS signup_requests (
	id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	business_name  TEXT NOT NULL,
	contact_name   TEXT NOT NULL,
	email          TEXT NOT NULL,
	phone          TEXT,
	website        TEXT,
	country        TEXT,
	description    TEXT,
	expected_volume TEXT,               -- champ libre ("~500 000 F/mois")
	status         TEXT NOT NULL DEFAULT 'pending'
		CHECK (status IN ('pending', 'approved', 'rejected')),
	review_note    TEXT,                -- motif de rejet ou note d'approbation
	app_id         UUID REFERENCES apps(id),  -- rempli à l'approbation
	created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
	reviewed_at    TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_signup_requests_status
	ON signup_requests (status, created_at DESC);

-- Anti-spam léger : une seule demande "pending" par email à la fois
-- (l'admin traite avant qu'un renvoi soit possible).
CREATE UNIQUE INDEX IF NOT EXISTS uq_signup_pending_email
	ON signup_requests (lower(email)) WHERE status = 'pending';
