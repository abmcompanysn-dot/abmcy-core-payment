// Package auth — génération/hachage de jetons opaques pour ABMCY Core,
// même principe que DIARRA backend/internal/auth/token.go (dupliqué plutôt
// qu'importé : modules Go séparés, ABMCY Core ne dépend jamais du code
// interne de DIARRA, seulement de son API HTTP publique).
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

// GenerateToken produit un jeton opaque aléatoire (32 octets, hex).
func GenerateToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// HashToken calcule l'empreinte SHA-256 d'un jeton. Seule l'empreinte est
// stockée en base — un jeton volé dans la base ne peut pas être réutilisé.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
