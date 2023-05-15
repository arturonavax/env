# Common personal terminal commands

## Search or replacement

Cambiar permiso a todos los archivos desde un lugar :

```bash
find . -type f -exec chmod 644 {} \;
```

Cambiar texto en todos los archivos desde un lugar:

```bash
find . -type f -exec sed -i 's/text/other/g' {} \;
```

Buscar texto en todos los archivos desde un lugar:

```bash
grep -rni 'text' .
```

## Monitoring / System administration

Ver todos los archivos abiertos por un proceso:

```bash
lsof -c <processName>
# or
lsof -p <PID>
```

Ejecutar un comando cada 1 segundo (`-n 1`) resaltando diferencias (`-d`):

```bash
watch -n 1 -d "<command>"
```

Ejecutar comando con `screen`:

```bash
screen -dm <command>
```

Ver instancias de `screen`:

```bash
screen -ls
```

Retomar instancia de `screen`:

```bash
screen -r
```

## Develop

Actualizar dependencias de un proyecto en Go:

```bash
go mod tidy && go get -u && go build
```

Escanear vulnerabilidades o problemas de seguridad de un proyecto en Go:

```bash
govulncheck ./... ; gosec ./... ; go list -json -deps | nancy sleuth
```

Contar todas las linea de un proyecto (agregar a los `--exclude` / `--exclude-dir`
los archivos y carpetas que no les interesa contar):

```bash
grep -r -I -l \
  --exclude-dir="node_modules" \
  --exclude-dir="vendor" \
  --exclude-dir="dist" \
  --exclude-dir="testdata" \
  --exclude-dir=".git" \
  --exclude-dir=".github" \
  --exclude="package.json" \
  --exclude="package-lock.json" \
  --exclude="*.svg" \
  --exclude="*.map" \
  --exclude="*.sum" \
  --exclude="*.lock" \
  --exclude="*.log" \
  --exclude="LICENSE" \
  --exclude=".gitignore" . \
  | xargs wc -l
```

Copiar el path actual al portapapeles:

```bash
pwd | xclip - selection clipboard:
# alias "cpath"
```

Crear un archivo pesado:

```bash
fallocate -l 2G big_file.txt

# or (takes a little longer)
dd if=/dev/zero bs=4k iflag=fullblock,count_bytes count=2G of=./big_file.txt

# or (without actually using up disk space)
truncate -s 2G big_file.txt
```

## Others

Actualizar totalmente el sistema:

```bash
sudo apt -y update && sudo apt -y upgrade && sudo apt -y dist-upgrade &&
  sudo apt -y full-upgrade && sudo snap refresh && sudo apt autoremove &&
  sudo apt autoclean
# alias "updateall"
```

Limpiar swap:

```bash
sudo su -c "sync; echo 3 > /proc/sys/vm/drop_caches ; swapoff -a && swapon -a"
# alias "cacheclear"
```

Limpiar cache de DNS:

```bash
sudo systemd-resolve --flush-caches
# alias "flushdns"
```

Calcular fechas:

```bash
date -d "13 Sep 2021 + 180 days"
```

Obtener tu IP Publica:

```bash
curl -s ipinfo.io/ip
# alias "publicip"
```

Obtener la geolocalizacion de una IP Publica:

```bash
curl ipinfo.io/{PublicIP}
# example: curl ipinfo.io/1.0.0.1
```

Geolocalizar tu IP Publica:

```bash
curl ipinfo.io/"$(curl -s ipinfo.io/ip)"
# alias "geoip"
```

Obtener Gateway IP:

```bash
route -n
```
