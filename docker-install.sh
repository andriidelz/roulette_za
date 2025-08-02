yum remove docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin docker-ce-rootless-extras
rm -rf /var/lib/docker /var/lib/containerd

yum install -y yum-utils
yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo

yum install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

systemctl daemon-reexec
systemctl daemon-reload
systemctl start docker
systemctl enable docker

docker version
docker info
