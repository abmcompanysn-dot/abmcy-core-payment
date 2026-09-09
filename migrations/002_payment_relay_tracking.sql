-- Migration 002 : suivi du relais ABMCY Core -> app cliente.
--
-- DiarraCallback met à jour un `payments` puis relaie l'événement vers le
-- `callback_url` de l'app (voir PaymentHandler.relayToApp). On garde une
-- trace du dernier relais pour que le dashboard admin puisse afficher
-- "livré / échoué" et proposer un renvoi manuel — même besoin que la page
-- /admin/payouts côté DIARRA (un relais raté doit être rattrapable à la
-- main, jamais silencieux).

ALTER TABLE payments
	ADD COLUMN IF NOT EXISTS relay_status TEXT NOT NULL DEFAULT 'pending'
		CHECK (relay_status IN ('pending', 'delivered', 'failed', 'skipped')),
	ADD COLUMN IF NOT EXISTS relay_attempts INTEGER NOT NULL DEFAULT 0,
	ADD COLUMN IF NOT EXISTS relay_last_error TEXT,
	ADD COLUMN IF NOT EXISTS relay_last_attempt_at TIMESTAMPTZ;
