-- Migration 003 : return_url pour la page de paiement hébergée.
--
-- Quand une app utilise la page hébergée core.diarra.app/pay/{ref} (ou le
-- widget), elle passe une return_url : l'URL de SA boutique où renvoyer
-- l'utilisateur une fois le paiement terminé (succès ou échec). Distincte
-- de callback_url (notification serveur-à-serveur) — celle-ci est pour le
-- navigateur de l'utilisateur final.

ALTER TABLE payments
	ADD COLUMN IF NOT EXISTS return_url TEXT;
