FROM alpine:3.23
RUN apk add --no-cache samba-server samba-common-tools \
    && adduser -D docflow \
    && mkdir -p /shares/backups /run/samba \
    && chown docflow:docflow /shares/backups \
    && printf 'integration-password\nintegration-password\n' | smbpasswd -a -s docflow
COPY build/integration-smb/smb.conf /etc/samba/smb.conf
CMD ["smbd", "--foreground", "--no-process-group", "--debug-stdout"]
