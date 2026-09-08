#!/bin/sh
# Déploie ABMCY Core Payment sur le VPS diarra-vps (k3s, namespace
# abmcy-core). Même principe que DIARRA k8s/deploy.sh : build local, import
# dans containerd (pas de registre), rollout forcé (tag :latest immuable).
#
# Prérequis (une seule fois) :
#   - base abmcy_core créée sur le Postgres du namespace diarra
#   - secret abmcy-core-secrets créé (voir k8s/secret.example.yaml)
#   - DNS core.diarra.app -> IP du VPS (Cloudflare proxied, SSL Full)
#   - bloc Caddy pour core.diarra.app (voir k8s/Caddyfile.snippet)
#
# À lancer depuis la racine du dépôt, SUR le VPS.
set -eu
export KUBECONFIG=/etc/rancher/k3s/k3s.yaml

echo "== Build image =="
timeout 480 docker build -t abmcy-core:latest .

echo "== Import dans containerd (k3s) =="
docker save abmcy-core:latest | k3s ctr images import -

echo "== Application des manifests =="
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/ingress.yaml

echo "== Redémarrage forcé (nouvelle image, même tag) =="
kubectl -n abmcy-core rollout restart deployment/abmcy-core

echo "== Attente =="
kubectl -n abmcy-core rollout status deployment/abmcy-core --timeout=120s

echo "== Terminé =="
kubectl -n abmcy-core get pods
