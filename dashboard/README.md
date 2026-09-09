# ABMCY Core Payment — Console

Interface d'administration de l'orchestrateur [ABMCY Core Payment](https://github.com/abmcompanysn-dot/abmcy-core-payment).
App Next.js séparée : elle ne parle qu'à l'API `/admin/*` d'ABMCY Core.

## Principe

- Connexion par le **jeton admin** d'ABMCY Core (`ABMCY_ADMIN_TOKEN`). Le jeton
  est vérifié côté serveur puis stocké dans un **cookie httpOnly** — il ne
  transite jamais par le JavaScript du navigateur.
- Le navigateur n'appelle que les API routes Next (`/api/*`), qui relaient vers
  `core.diarra.app` avec l'en-tête `Authorization: Bearer`. Aucune config CORS
  nécessaire sur ABMCY Core.

## Écrans

| Route | Rôle |
|---|---|
| `/login` | Saisie du jeton admin |
| `/payments` | Liste des paiements (filtres app / statut), statut + statut de relais |
| `/payments/{id}` | Détail d'un paiement, historique de relais, **renvoi manuel du relais** |
| `/apps` | Liste des applications clientes, création (clé + secret affichés une fois), activation/désactivation |

## Dév local

```bash
npm install
cp .env.example .env.local   # ajuster ABMCY_CORE_URL si besoin
npm run dev                   # http://localhost:3100
```

## Configuration

| Variable | Défaut | Rôle |
|---|---|---|
| `ABMCY_CORE_URL` | `https://core.diarra.app` | Base de l'API ABMCY Core |

## Build

```bash
npm run build && npm start
```
