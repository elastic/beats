FROM mcr.microsoft.com/windows:10.0.19041.746-amd64 AS vm_extension_windows

WORKDIR /sln
USER ContainerAdministrator
RUN NET USER my_admin /add
RUN	NET LOCALGROUP Administrators /add my_admin
USER my_admin
COPY ./handler ./handler
