# vim: sw=4:ts=4:et
%define relabel_files() \
restorecon -R /%{_lib}/systemd/system;                \
restorecon -R %{_datadir}/filebeat/bin;               \
restorecon -R %{_datadir}/journalbeat/bin;            \
restorecon -R %{_datadir}/auditbeat/bin;              \
restorecon -R %{_bindir};                             \
restorecon -R %{_sharedstatedir}/auditbeat;           \
restorecon -R %{_localstatedir}/log/auditbeat;        \
restorecon -R %{_sharedstatedir}/filebeat;            \
restorecon -R %{_localstatedir}/log/filebeat;         \
restorecon -R %{_sharedstatedir}/journalbeat;         \
restorecon -R %{_localstatedir}/log/journalbeat;

Name:               beats-selinux
Version:            1.0
Release:            1%{?dist}
Summary:            SELinux policy module for various beats

Group:              System Environment/Base     
License:            GPLv2+  
URL:                https://git.im.jku.at/summary/packages!beats-selinux.git
Source0:            beats.te
Source1:            beats.if
Source2:            filebeat.te
Source3:            filebeat.fc
Source4:            filebeat.if
Source5:            auditbeat.te
Source6:            auditbeat.fc
Source7:            auditbeat.if
Source8:            journalbeat.te
Source9:            journalbeat.fc
Source10:           journalbeat.if

BuildRequires:      selinux-policy-devel >= 3.13
BuildConflicts:     selinux-policy-devel < 3.13
BuildRequires:      policycoreutils-devel
Requires:           policycoreutils
Requires:           libselinux-utils
Requires:           selinux-policy >= 3.13
Conflicts:          selinux-policy < 3.13
Requires(post):     policycoreutils, policycoreutils-python 
Requires(postun):     policycoreutils, policycoreutils-python 
BuildArch:          noarch

%description
This package installs and sets up the SELinux policy security module for beats.

%prep
%setup -c -n %{name} -T
cp %{SOURCE0} %{SOURCE1} %{SOURCE2} %{SOURCE3} \
   %{SOURCE4} %{SOURCE5} %{SOURCE6} %{SOURCE7} \
   %{SOURCE8} %{SOURCE9} %{SOURCE10} \
 .

%build
make -f /usr/share/selinux/devel/Makefile beats.pp || exit
make -f /usr/share/selinux/devel/Makefile filebeat.pp || exit
make -f /usr/share/selinux/devel/Makefile journalbeat.pp || exit
make -f /usr/share/selinux/devel/Makefile auditbeat.pp || exit

%install
install -d %{buildroot}%{_datadir}/selinux/packages
install -m 644 beats.pp %{buildroot}%{_datadir}/selinux/packages
install -m 644 filebeat.pp %{buildroot}%{_datadir}/selinux/packages
install -m 644 journalbeat.pp %{buildroot}%{_datadir}/selinux/packages
install -m 644 auditbeat.pp %{buildroot}%{_datadir}/selinux/packages
install -d %{buildroot}%{_datadir}/selinux/devel/include/contrib
install -m 644 filebeat.if %{buildroot}%{_datadir}/selinux/devel/include/contrib/
install -m 644 journalbeat.if %{buildroot}%{_datadir}/selinux/devel/include/contrib/
install -m 644 auditbeat.if %{buildroot}%{_datadir}/selinux/devel/include/contrib/
install -m 644 beats.if %{buildroot}%{_datadir}/selinux/devel/include/contrib/
install -d %{buildroot}/etc/selinux/targeted/contexts/users/

%post
semodule -n -i %{_datadir}/selinux/packages/beats.pp
semodule -n -i %{_datadir}/selinux/packages/filebeat.pp
semodule -n -i %{_datadir}/selinux/packages/journalbeat.pp
semodule -n -i %{_datadir}/selinux/packages/auditbeat.pp

if /usr/sbin/selinuxenabled ; then
    /usr/sbin/load_policy
    %relabel_files
fi;

semanage port -p tcp -t logstash_port_t -a 5044
semanage port -p tcp -t kafka_port_t -a 9092
semanage port -p tcp -t elasticsearch_port_t -a 9200
exit 0
 
%postun
if [ $1 -eq 0 ]; then
    semanage port -p tcp -t logstash_port_t -d 5044
    semanage port -p tcp -t kafka_port_t -d 9092
    semanage port -p tcp -t elasticsearch_port_t -d 9200

    semodule -n -r filebeat
    semodule -n -r journalbeat
    semodule -n -r auditbeat
    semodule -n -r beats

    if /usr/sbin/selinuxenabled ; then
       /usr/sbin/load_policy
       %relabel_files
    fi;
fi;
exit 0

%files
%defattr(-,root,root,-)
%{_datadir}/selinux/packages/beats.pp
%{_datadir}/selinux/packages/filebeat.pp
%{_datadir}/selinux/packages/journalbeat.pp
%{_datadir}/selinux/packages/auditbeat.pp
%{_datadir}/selinux/devel/include/contrib/beats.if
%{_datadir}/selinux/devel/include/contrib/filebeat.if
%{_datadir}/selinux/devel/include/contrib/journalbeat.if
%{_datadir}/selinux/devel/include/contrib/auditbeat.if

%changelog
* Wed Jan 17 2018 fuero <fuerob@gmail.com> - 1.0-1
- Initial version

