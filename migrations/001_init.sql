-- Migration 001 : fondations multi-application d'ABMCY Core Payment.
--
-- ABMCY Core est un ORCHESTRATEUR : plusieurs applications ABMCY (une par
-- produit) intègrent son widget/SDK de paiement, et c'est ABMCY Core qui
-- route chaque paiement vers la passerelle DIARRA (une seule connexion
-- agrégateur pour tout l'écosystème ABMCY, voir DIARRA
-- backend/internal/payment/provider.go). Chaque application cliente a donc
-- besoin de SA PROPRE identité ici — exactement le même principe que
-- gateway_clients côté DIARRA, mais un niveau au-dessus.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Une application ABMCY qui intègre le widget/SDK de paiement (ex. une
-- boutique, une app de réservation...). api_key_hash/hmac_secret_hash :
-- jamais en clair, même principe que DIARRA gateway_clients.
CREATE TABLE IF NOT EXISTS apps (
	id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	name              TEXT NOT NULL,
	api_key_hash      TEXT NOT NULL UNIQUE,
	hmac_secret_hash  TEXT NOT NULL,
	default_callback_url TEXT,
	is_active         BOOLEAN NOT NULL DEFAULT TRUE,
	created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Un paiement initié par une app cliente via ABMCY Core. app_ref est
-- l'identifiant CHEZ L'APP (sa propre commande), jamais réutilisé ici — même
-- principe que client_ref côté DIARRA gateway_transactions, un niveau plus
-- haut : ABMCY Core relie app_ref (app -> ABMCY Core) à diarra_client_ref
-- (ABMCY Core -> DIARRA, généré ici, unique par transaction).
CREATE TABLE IF NOT EXISTS payments (
	id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	app_id              UUID NOT NULL REFERENCES apps(id),
	app_ref             TEXT NOT NULL,
	diarra_client_ref   TEXT NOT NULL UNIQUE,
	type                TEXT NOT NULL DEFAULT 'deposit'
		CHECK (type IN ('deposit', 'payout', 'refund')),
	provider            TEXT, -- copié depuis la transaction DIARRA une fois connue (pawapay|kpay|paypal)
	status              TEXT NOT NULL DEFAULT 'pending'
		CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'cancelled')),
	failure_reason      TEXT,
	amount_cfa          INTEGER NOT NULL,
	currency            TEXT NOT NULL DEFAULT 'XOF',
	description         TEXT,
	redirect_url        TEXT,
	callback_url        TEXT, -- URL de l'app à notifier (relais ABMCY Core -> app)
	created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),

	UNIQUE (app_id, app_ref)
);

CREATE INDEX IF NOT EXISTS idx_payments_app ON payments (app_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_payments_diarra_ref ON payments (diarra_client_ref);
