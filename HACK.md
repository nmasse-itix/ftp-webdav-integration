# Hacking

## FTP

vsftpd ?

Active vs Passive: https://fr.wikipedia.org/wiki/File_Transfer_Protocol

Active: le serveur FTP initie la connexion données (et ça pose problème avec le NAT).
Passive: le client FTP initie la connexion données (et ça demande d'ouvrir une plage de ports coté serveur).

Sur vsftpd, il est possible de spécifier la plage de port pour le mode passif.

```
pasv_enable=Yes
pasv_max_port=10100
pasv_min_port=10090
```


## Camel

https://camel.apache.org/components/3.18.x/ftp-component.html
https://camel.apache.org/components/3.19.x/ftps-component.html
but no webdav component

## Go

https://github.com/studio-b12/gowebdav
https://github.com/secsy/goftp

