Summary:	Packetbeat network agent
Name:		packetbeat
Version:	0.1.0
Release:	1%{?dist}
Source:		%{name}.tar.gz
Group:		Network
License:	GPLv2
URL:		http://packetbeat.com

Requires:	libpcap, daemonize

%description
Packetbeat Agent

%prep
%setup -n %{name}

%build
make

%install
make install DESTDIR=%{buildroot}

%files
/usr/bin/*

%doc

%changelog

