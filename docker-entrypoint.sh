#!/bin/sh
set -e

echo "== Application des migrations (base abmcy_core) =="
./migrate

echo "== ABMCY Core : port ${PORT:-9090} =="
exec ./server
