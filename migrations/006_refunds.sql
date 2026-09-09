-- Migration 006 : remboursements.
--
-- Un remboursement est un `payments` de type 'refund' qui référence le
-- paiement d'origine (refund_of_payment_id). L'app le déclenche via son
-- propre app_ref du dépôt initial ; ABMCY Core retrouve le
-- diarra_client_ref correspondant et le passe à la passerelle DIARRA.
--
-- La contrainte UNIQUE(app_id, app_ref) existante interdirait deux lignes
-- avec le même app_ref (le dépôt ET son remboursement). On la remplace par
-- une unicité qui inclut le type.

ALTER TABLE payments
	ADD COLUMN IF NOT EXISTS refund_of_payment_id UUID REFERENCES payments(id);

ALTER TABLE payments DROP CONSTRAINT IF EXISTS payments_app_id_app_ref_key;

CREATE UNIQUE INDEX IF NOT EXISTS uq_payments_app_ref_type
	ON payments (app_id, app_ref, type);

CREATE INDEX IF NOT EXISTS idx_payments_refund_of
	ON payments (refund_of_payment_id);
