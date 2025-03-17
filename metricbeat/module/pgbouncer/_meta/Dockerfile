FROM edoburu/pgbouncer
COPY ./pgbouncer.ini /etc/pgbouncer/pgbouncer.ini
COPY ./userlist.txt /etc/pgbouncer/userlist.txt
ENTRYPOINT ["/entrypoint.sh"]
CMD ["/usr/bin/pgbouncer", "/etc/pgbouncer/pgbouncer.ini"]