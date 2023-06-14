#!/bin/bash
set -ex

apt update -y
apt install -y apache2
systemctl enable apache2
mkdir -p /var/www/html/ubuntu
chown www-data:www-data /var/www/html/ubuntu
apt install -y apt-mirror
cp /etc/apt/mirror.list /etc/apt/mirror.list-bak


cat >/etc/apt/mirror.list <<EOL
############# config ###################
set base_path    /var/www/html/ubuntu
set nthreads     20
set _tilde 0
############# end config ##############
deb http://archive.ubuntu.com/ubuntu focal main restricted universe \
 multiverse
deb http://archive.ubuntu.com/ubuntu focal-security main restricted \
universe multiverse
deb http://archive.ubuntu.com/ubuntu focal-updates main restricted \
universe multiverse
clean http://archive.ubuntu.com/ubuntu
EOL

mkdir -p /var/www/html/ubuntu/var
cp /var/spool/apt-mirror/var/postmirror.sh /var/www/html/ubuntu/var

nohup apt-mirror &

for p in "${1:-focal}"{,-{security,updates}}\
/{main,restricted,universe,multiverse};do >&2 echo "${p}"
wget -q -c -r -np -R "index.html*" "http://archive.ubuntu.com/ubuntu/dists/${p}/cnf/Commands-amd64.xz"
wget -q -c -r -np -R "index.html*" "http://archive.ubuntu.com/ubuntu/dists/${p}/cnf/Commands-i386.xz"
done

cp -av archive.ubuntu.com  /var/www/html/ubuntu/mirror/

ufw allow 80
