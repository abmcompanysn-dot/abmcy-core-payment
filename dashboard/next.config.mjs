/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  // Servi derrière core.diarra.app/console (voir l'ingress d'ABMCY Core).
  // basePath préfixe routes + assets. En dev local, poser
  // NEXT_PUBLIC_BASE_PATH="" pour revenir à la racine (http://localhost:3100).
  basePath: process.env.NEXT_PUBLIC_BASE_PATH ?? '/console',
  output: 'standalone',
};

export default nextConfig;
