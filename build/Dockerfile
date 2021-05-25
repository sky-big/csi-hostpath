FROM centos
LABEL maintainers="Kubernetes CSI HOSTPATH"
LABEL description="Kubernetes CSI HOSTPATH Driver"

# add nsenter
COPY nsenter /nsenter

# add plugin
COPY _output/hostpath.csi.kubernetes.io /bin/hostpath.csi.kubernetes.io
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /bin/hostpath.csi.kubernetes.io && chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]