# Serveur
Écoute sur le port **2000** par défaut.

Il essaiera par défaut de se connecter à un proxy inverse pour récupérer des clients hors du LAN.

Attention, le reverse proxy enverra tous les clients au premier serveur qui le contacte!
# Client
Il se connecte au serveur et donne l'URL du site à vérifier (sans `/` à la fin).
Le client ne fait pas de différence entre un serveur et un proxy inverse, le dialogue est le même.

Le dialogue client-serveur est extrêmement strict et ne souffre aucun écart au protocole.
# Proxy inverse
Il permet à un client de dialoguer avec un serveur hors de son LAN (ou sur une adresse non routable).
L'option `-no-proxy` désactive la fonction de proxy inverse.